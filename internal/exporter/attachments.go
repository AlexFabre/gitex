package exporter

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// Document extensions to download to appendix.
var documentExts = map[string]bool{
	".pdf":   true,
	".docx":  true,
	".doc":   true,
	".xlsx":  true,
	".xls":   true,
	".pptx":  true,
	".ppt":   true,
	".csv":   true,
	".zip":   true,
	".drawio": true,
}

// linkRef matches markdown link syntax: [text](url) but NOT image syntax ![text](url)
var linkRef = regexp.MustCompile(`(?:^|[^!])\[([^\]]*)\]\(([^)]+)\)`)

// DownloadAttachments finds document links in markdown content, downloads them
// to appendixDir, and rewrites the links to point to local files. For .drawio
// files, it also attempts to render a PNG and inserts an image reference.
func DownloadAttachments(content, appendixDir, imagesDir, gitlabURL, projectPath, token string, issueIID int64, imgCounter *ImageCounter) (string, error) {
	matches := linkRef.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return content, nil
	}

	// Process in reverse to preserve indices during replacements.
	for i := len(matches) - 1; i >= 0; i-- {
		m := matches[i]
		linkText := content[m[2]:m[3]]
		linkURL := content[m[4]:m[5]]

		// Determine the file extension (decode first for percent-encoded names).
		decoded, err := url.PathUnescape(linkURL)
		if err != nil {
			decoded = linkURL
		}
		ext := strings.ToLower(filepath.Ext(decoded))
		if !documentExts[ext] {
			continue
		}

		if !shouldDownload(linkURL, gitlabURL) {
			continue
		}

		if err := os.MkdirAll(appendixDir, 0o755); err != nil {
			return "", fmt.Errorf("creating appendix directory: %w", err)
		}

		absURL := resolveURL(linkURL, gitlabURL, projectPath)

		// Build a filename: issue-<IID>-<original-basename>
		baseName := filepath.Base(decoded)
		filename := fmt.Sprintf("issue-%d-%s", issueIID, baseName)

		localPath, err := downloadFile(absURL, appendixDir, token, filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to download attachment %s: %v\n", linkURL, err)
			continue
		}

		// Compute relative path from the markdown file to the appendix.
		relPath := filepath.Join("../appendix", filepath.Base(localPath))

		// Find the actual start of the markdown link (skip the non-! char captured by the lookbehind).
		linkStart := m[0]
		if linkStart < m[2]-1 {
			// The regex captured a char before '[', adjust to start at '['
			linkStart = m[2] - 1
		}

		replacement := fmt.Sprintf("[%s](%s)", linkText, relPath)

		// For .drawio files, try to render a PNG image.
		if ext == ".drawio" {
			pngPath, renderErr := renderDrawio(localPath, imagesDir, issueIID, imgCounter)
			if renderErr == nil {
				imgRelPath := filepath.Join("images", filepath.Base(pngPath))
				replacement = fmt.Sprintf("![%s](%s)\n\n[%s (source)](%s)", linkText, imgRelPath, linkText, relPath)
			}
		}

		content = content[:linkStart] + replacement + content[m[1]:]
	}

	return content, nil
}

// downloadFile is the same as downloadImage but without image-specific logic.
func downloadFile(fileURL, destDir, token, filename string) (string, error) {
	return downloadImage(fileURL, destDir, token, filename)
}

// renderDrawio converts a .drawio file to PNG using the drawio CLI.
// Returns the output PNG path or an error if drawio is not available.
func renderDrawio(drawioPath, imagesDir string, issueIID int64, counter *ImageCounter) (string, error) {
	drawioCmd := findDrawioCmd()
	if drawioCmd == "" {
		return "", fmt.Errorf("drawio CLI not found")
	}

	if err := os.MkdirAll(imagesDir, 0o755); err != nil {
		return "", fmt.Errorf("creating images directory: %w", err)
	}

	imgNum := counter.Next()
	outputFile := filepath.Join(imagesDir, fmt.Sprintf("issue-%d-image-%d.png", issueIID, imgNum))

	cmd := exec.Command(drawioCmd, "-x", "-f", "png", "-o", outputFile, drawioPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("rendering drawio: %w: %s", err, string(output))
	}

	return outputFile, nil
}

// findDrawioCmd locates the draw.io CLI binary. It checks PATH first, then
// well-known installation locations on macOS and Linux.
func findDrawioCmd() string {
	for _, name := range []string{"drawio", "draw.io"} {
		if p, err := exec.LookPath(name); err == nil {
			return p
		}
	}
	return ""
}
