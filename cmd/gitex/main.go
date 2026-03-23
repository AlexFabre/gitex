package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/AlexFabre/gitex/internal/config"
	"github.com/AlexFabre/gitex/internal/exporter"
	"github.com/AlexFabre/gitex/internal/gitlab"
)

var version = "dev"

func main() {
	rootCmd := &cobra.Command{
		Use:     "gitex",
		Short:   "Extract content from GitLab and generate local markdown documents",
		Long:    `gitlex is a CLI tool that fetches issues, merge requests, and other
content from a GitLab repository and generates local markdown documents.`,
		Version: version,
	}

	config.BindFlags(rootCmd)

	issuesCmd := &cobra.Command{
		Use:   "issues",
		Short: "Fetch GitLab issues and export them as markdown",
		RunE:  runIssues,
	}
	config.BindIssuesFlags(issuesCmd)
	rootCmd.AddCommand(issuesCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runIssues(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadIssues()
	if err != nil {
		return err
	}

	client, err := gitlab.NewClient(cfg.GitLabURL, cfg.Token, cfg.Project)
	if err != nil {
		return err
	}

	if cfg.IssueID > 0 {
		return fetchSingleIssue(client, cfg)
	}

	return fetchAllIssues(client, cfg)
}

func fetchSingleIssue(client *gitlab.Client, cfg *config.IssuesConfig) error {
	fmt.Printf("Fetching issue #%d from %s...\n", cfg.IssueID, cfg.Project)

	issue, err := client.GetIssue(cfg.IssueID)
	if err != nil {
		return err
	}

	path, err := exporter.ExportIssue(issue, cfg.Output, cfg.GitLabURL, cfg.Project, cfg.Token)
	if err != nil {
		return err
	}

	fmt.Printf("Exported: %s\n", path)
	return nil
}

func fetchAllIssues(client *gitlab.Client, cfg *config.IssuesConfig) error {
	fmt.Printf("Fetching issues from %s...\n", cfg.Project)

	issues, err := client.ListIssues(cfg.State, cfg.Labels)
	if err != nil {
		return err
	}

	if len(issues) == 0 {
		fmt.Println("No issues found.")
		return nil
	}

	fmt.Printf("Found %d issues. Exporting...\n", len(issues))

	for _, issue := range issues {
		path, err := exporter.ExportIssue(issue, cfg.Output, cfg.GitLabURL, cfg.Project, cfg.Token)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to export issue #%d: %v\n", issue.IID, err)
			continue
		}
		fmt.Printf("  Exported: %s\n", path)
	}

	fmt.Printf("Done. %d issues exported to %s\n", len(issues), cfg.Output)
	return nil
}
