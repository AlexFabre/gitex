package exporter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AlexFabre/gitex/internal/gitlab"
)

// ExportIssue writes a single issue to a markdown file, downloads its images
// to issuesDir/images/, and downloads document attachments to outputDir/appendix/.
// issuesDir is the flat directory where all issue markdown files are written
// (e.g. output/issues/). outputDir is the parent (e.g. output/).
func ExportIssue(issue *gitlab.Issue, outputDir, gitlabURL, projectPath, token string) (string, error) {
	issuesDir := filepath.Join(outputDir, "issues")
	if err := os.MkdirAll(issuesDir, 0o755); err != nil {
		return "", fmt.Errorf("creating issues directory: %w", err)
	}

	imagesDir := filepath.Join(issuesDir, "images")
	appendixDir := filepath.Join(outputDir, "appendix")
	counter := &ImageCounter{}

	// Process images in description.
	description, err := DownloadImages(issue.Description, imagesDir, gitlabURL, projectPath, token, issue.IID, counter)
	if err != nil {
		return "", fmt.Errorf("processing description images: %w", err)
	}

	// Process document attachments in description.
	description, err = DownloadAttachments(description, appendixDir, imagesDir, gitlabURL, projectPath, token, issue.IID, counter)
	if err != nil {
		return "", fmt.Errorf("processing description attachments: %w", err)
	}

	// Process images and attachments in notes.
	var processedNotes []gitlab.Note
	for _, note := range issue.Notes {
		body, err := DownloadImages(note.Body, imagesDir, gitlabURL, projectPath, token, issue.IID, counter)
		if err != nil {
			return "", fmt.Errorf("processing note images: %w", err)
		}
		body, err = DownloadAttachments(body, appendixDir, imagesDir, gitlabURL, projectPath, token, issue.IID, counter)
		if err != nil {
			return "", fmt.Errorf("processing note attachments: %w", err)
		}
		processedNotes = append(processedNotes, gitlab.Note{
			Author:    note.Author,
			Body:      body,
			CreatedAt: note.CreatedAt,
		})
	}

	md := renderIssueMarkdown(issue, description, processedNotes)

	filePath := filepath.Join(issuesDir, fmt.Sprintf("issue-%d.md", issue.IID))
	if err := os.WriteFile(filePath, []byte(md), 0o644); err != nil {
		return "", fmt.Errorf("writing markdown file: %w", err)
	}

	return filePath, nil
}

func renderIssueMarkdown(issue *gitlab.Issue, description string, notes []gitlab.Note) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# #%d — %s\n\n", issue.IID, issue.Title)

	fmt.Fprintf(&b, "- **State**: %s\n", issue.State)
	fmt.Fprintf(&b, "- **Author**: %s\n", issue.Author)
	fmt.Fprintf(&b, "- **Created**: %s\n", issue.CreatedAt)

	if issue.ClosedAt != "" {
		fmt.Fprintf(&b, "- **Closed**: %s\n", issue.ClosedAt)
	}

	if len(issue.Labels) > 0 {
		fmt.Fprintf(&b, "- **Labels**: %s\n", strings.Join(issue.Labels, ", "))
	}

	if len(issue.Assignees) > 0 {
		fmt.Fprintf(&b, "- **Assignees**: %s\n", strings.Join(issue.Assignees, ", "))
	}

	if issue.Milestone != "" {
		fmt.Fprintf(&b, "- **Milestone**: %s\n", issue.Milestone)
	}

	fmt.Fprintf(&b, "- **URL**: %s\n", issue.WebURL)

	b.WriteString("\n---\n\n")
	b.WriteString("## Description\n\n")

	if description != "" {
		b.WriteString(description)
	} else {
		b.WriteString("*No description provided.*")
	}

	b.WriteString("\n")

	if len(notes) > 0 {
		b.WriteString("\n---\n\n")
		b.WriteString("## Comments\n\n")

		for _, note := range notes {
			fmt.Fprintf(&b, "### %s — %s\n\n", note.Author, note.CreatedAt)
			b.WriteString(note.Body)
			b.WriteString("\n\n")
		}
	}

	return b.String()
}
