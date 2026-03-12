package gitlab

import (
	"fmt"

	gogitlab "gitlab.com/gitlab-org/api/client-go"
)

// Client wraps the GitLab API client.
type Client struct {
	client  *gogitlab.Client
	project string
}

// Issue holds the data we care about for an issue.
type Issue struct {
	IID         int64
	Title       string
	State       string
	Description string
	Author      string
	Assignees   []string
	Labels      []string
	Milestone   string
	CreatedAt   string
	ClosedAt    string
	WebURL      string
	Notes       []Note
}

// Note represents a comment on an issue.
type Note struct {
	Author    string
	Body      string
	CreatedAt string
}

// NewClient creates a new GitLab API client.
func NewClient(baseURL, token, project string) (*Client, error) {
	client, err := gogitlab.NewClient(token, gogitlab.WithBaseURL(baseURL+"/api/v4"))
	if err != nil {
		return nil, fmt.Errorf("creating gitlab client: %w", err)
	}

	return &Client{
		client:  client,
		project: project,
	}, nil
}

// GetIssue fetches a single issue by IID.
func (c *Client) GetIssue(iid int64) (*Issue, error) {
	issue, _, err := c.client.Issues.GetIssue(c.project, iid)
	if err != nil {
		return nil, fmt.Errorf("fetching issue #%d: %w", iid, err)
	}

	result := convertIssue(issue)

	notes, err := c.getIssueNotes(iid)
	if err != nil {
		return nil, err
	}
	result.Notes = notes

	return result, nil
}

// ListIssues fetches all issues matching the given filters.
func (c *Client) ListIssues(state string, labels []string) ([]*Issue, error) {
	var allIssues []*Issue

	opts := &gogitlab.ListProjectIssuesOptions{
		ListOptions: gogitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
	}

	if state != "" {
		opts.State = gogitlab.Ptr(state)
	}
	if len(labels) > 0 {
		lbls := gogitlab.LabelOptions(labels)
		opts.Labels = &lbls
	}

	for {
		issues, resp, err := c.client.Issues.ListProjectIssues(c.project, opts)
		if err != nil {
			return nil, fmt.Errorf("listing issues: %w", err)
		}

		for _, issue := range issues {
			converted := convertIssue(issue)

			notes, err := c.getIssueNotes(issue.IID)
			if err != nil {
				return nil, err
			}
			converted.Notes = notes

			allIssues = append(allIssues, converted)
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allIssues, nil
}

func (c *Client) getIssueNotes(iid int64) ([]Note, error) {
	var allNotes []Note

	opts := &gogitlab.ListIssueNotesOptions{
		ListOptions: gogitlab.ListOptions{
			PerPage: 100,
			Page:    1,
		},
		Sort:    gogitlab.Ptr("asc"),
		OrderBy: gogitlab.Ptr("created_at"),
	}

	for {
		notes, resp, err := c.client.Notes.ListIssueNotes(c.project, iid, opts)
		if err != nil {
			return nil, fmt.Errorf("fetching notes for issue #%d: %w", iid, err)
		}

		for _, note := range notes {
			if note.System {
				continue // skip system notes (label changes, assignments, etc.)
			}
			allNotes = append(allNotes, Note{
				Author:    note.Author.Username,
				Body:      note.Body,
				CreatedAt: note.CreatedAt.Format("2006-01-02 15:04"),
			})
		}

		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allNotes, nil
}

func convertIssue(issue *gogitlab.Issue) *Issue {
	result := &Issue{
		IID:         issue.IID,
		Title:       issue.Title,
		State:       issue.State,
		Description: issue.Description,
		Labels:      issue.Labels,
		WebURL:      issue.WebURL,
	}

	if issue.Author != nil {
		result.Author = issue.Author.Username
	}

	for _, a := range issue.Assignees {
		result.Assignees = append(result.Assignees, a.Username)
	}

	if issue.Milestone != nil {
		result.Milestone = issue.Milestone.Title
	}

	if issue.CreatedAt != nil {
		result.CreatedAt = issue.CreatedAt.Format("2006-01-02")
	}

	if issue.ClosedAt != nil {
		result.ClosedAt = issue.ClosedAt.Format("2006-01-02")
	}

	return result
}
