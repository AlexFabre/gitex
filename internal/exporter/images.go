package exporter

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

const uploadsPath = "/uploads/"

// imageRef matches markdown image syntax: ![alt](url)
var imageRef = regexp.MustCompile(`!\[([^\]]*)\]\(([^)]+)\)`)

// ImageCounter tracks image numbering per issue across multiple calls.
type ImageCounter struct {
	mu      sync.Mutex
	counter int
}

// Next returns the next image number.
func (c *ImageCounter) Next() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.counter++
	return c.counter
}

// DownloadImages finds all image references in markdown content, downloads them
// to the given directory, and returns the content with rewritten local paths.
// projectPath is the GitLab project path (e.g. "group/project") used to resolve
// relative upload URLs. issueIID and counter are used to generate unique filenames
// like issue-3-image-4.png.
func DownloadImages(content, imagesDir, gitlabURL, projectPath, token string, issueIID int64, counter *ImageCounter) (string, error) {
	matches := imageRef.FindAllStringSubmatchIndex(content, -1)
	if len(matches) == 0 {
		return content, nil
	}

	if err := os.MkdirAll(imagesDir, 0o755); err != nil {
		return "", fmt.Errorf("creating images directory: %w", err)
	}

	// Process matches in reverse order so index positions remain valid after replacements.
	for i := len(matches) - 1; i >= 0; i-- {
		m := matches[i]
		alt := content[m[2]:m[3]]
		imgURL := content[m[4]:m[5]]

		// Only download images hosted on the GitLab instance or relative paths.
		if !shouldDownload(imgURL, gitlabURL) {
			continue
		}

		absURL := resolveURL(imgURL, gitlabURL, projectPath)

		// Determine the extension from the original URL.
		ext := filepath.Ext(filepath.Base(imgURL))
		imgNum := counter.Next()
		filename := fmt.Sprintf("issue-%d-image-%d%s", issueIID, imgNum, ext)

		localPath, err := downloadImage(absURL, imagesDir, token, filename)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to download image %s: %v\n", imgURL, err)
			continue
		}

		relPath := filepath.Join("images", filepath.Base(localPath))
		replacement := fmt.Sprintf("![%s](%s)", alt, relPath)
		content = content[:m[0]] + replacement + content[m[1]:]
	}

	return content, nil
}

func shouldDownload(imgURL, gitlabURL string) bool {
	// Relative URLs (GitLab uploads)
	if strings.HasPrefix(imgURL, uploadsPath) {
		return true
	}
	// Absolute URLs on the same GitLab instance
	if strings.HasPrefix(imgURL, gitlabURL) {
		return true
	}
	// Other absolute URLs — still download them
	if strings.HasPrefix(imgURL, "http://") || strings.HasPrefix(imgURL, "https://") {
		return true
	}
	return false
}

func resolveURL(imgURL, gitlabURL, projectPath string) string {
	base := strings.TrimRight(gitlabURL, "/")
	encodedProject := url.PathEscape(projectPath)
	apiPrefix := base + "/api/v4/projects/" + encodedProject

	// Relative upload path: /uploads/hash/file.png
	if strings.HasPrefix(imgURL, uploadsPath) {
		return apiPrefix + encodePathSegments(imgURL)
	}

	// Absolute URL on the same GitLab instance containing an upload path
	// e.g. https://git.example.com/group/project/uploads/hash/file.png
	if strings.HasPrefix(imgURL, base) {
		suffix := strings.TrimPrefix(imgURL, base)
		// Extract /uploads/... from the path (skip the project prefix)
		if idx := strings.Index(suffix, uploadsPath); idx >= 0 {
			return apiPrefix + encodePathSegments(suffix[idx:])
		}
	}

	// Other relative paths
	if strings.HasPrefix(imgURL, "/") {
		return base + encodePathSegments(imgURL)
	}

	return imgURL
}

// encodePathSegments percent-encodes each segment of a path individually,
// preserving slashes. It first decodes any existing percent-encoding to avoid
// double-encoding (e.g. %C3%A9 becoming %25C3%25A9).
func encodePathSegments(rawPath string) string {
	// Decode first to normalize: handles both raw UTF-8 and already-encoded paths.
	decoded, err := url.PathUnescape(rawPath)
	if err != nil {
		decoded = rawPath
	}
	segments := strings.Split(decoded, "/")
	for i, seg := range segments {
		segments[i] = url.PathEscape(seg)
	}
	return strings.Join(segments, "/")
}

// downloadImage downloads an image from imgURL, saves it to destDir with the
// given filename, and returns the full path. If the file has no extension but
// the response Content-Type suggests one, it is appended.
func downloadImage(imgURL, destDir, token, filename string) (string, error) {
	req, err := http.NewRequest("GET", imgURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	// Add auth token for GitLab-hosted images.
	if token != "" {
		req.Header.Set("PRIVATE-TOKEN", token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("downloading: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d for %s", resp.StatusCode, imgURL)
	}

	// Add extension from content type if missing.
	if filepath.Ext(filename) == "" {
		ct := resp.Header.Get("Content-Type")
		switch {
		case strings.Contains(ct, "png"):
			filename += ".png"
		case strings.Contains(ct, "jpeg"), strings.Contains(ct, "jpg"):
			filename += ".jpg"
		case strings.Contains(ct, "gif"):
			filename += ".gif"
		case strings.Contains(ct, "svg"):
			filename += ".svg"
		case strings.Contains(ct, "webp"):
			filename += ".webp"
		}
	}

	destPath := filepath.Join(destDir, filename)
	f, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("creating file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return "", fmt.Errorf("writing file: %w", err)
	}

	return destPath, nil
}
