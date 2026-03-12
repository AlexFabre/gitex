package config

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Config holds all configuration for the application.
type Config struct {
	GitLabURL string
	Token     string
	Project   string
	Output    string
}

// IssuesConfig holds configuration specific to the issues command.
type IssuesConfig struct {
	Config
	IssueID int64
	State   string
	Labels  []string
}

// BindFlags registers persistent flags on the root command and binds them to viper.
func BindFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String("gitlab-url", "https://gitlab.com", "GitLab instance URL")
	cmd.PersistentFlags().String("token", "", "GitLab private token")
	cmd.PersistentFlags().String("project", "", "GitLab project path (e.g. group/project)")
	cmd.PersistentFlags().String("output", "./output", "Output directory")

	viper.BindPFlag("gitlab_url", cmd.PersistentFlags().Lookup("gitlab-url"))
	viper.BindPFlag("token", cmd.PersistentFlags().Lookup("token"))
	viper.BindPFlag("project", cmd.PersistentFlags().Lookup("project"))
	viper.BindPFlag("output", cmd.PersistentFlags().Lookup("output"))

	viper.BindEnv("gitlab_url", "GITLAB_URL")
	viper.BindEnv("token", "GITLAB_TOKEN")
	viper.BindEnv("project", "GITLAB_PROJECT")
	viper.BindEnv("output", "GITLAB_OUTPUT")
}

// BindIssuesFlags registers flags specific to the issues command.
func BindIssuesFlags(cmd *cobra.Command) {
	cmd.Flags().Int64("issue-id", 0, "Specific issue ID to fetch (0 = all)")
	cmd.Flags().String("state", "", "Filter by state: opened, closed, all (default: all)")
	cmd.Flags().StringSlice("labels", nil, "Filter by labels (comma-separated)")

	viper.BindPFlag("issue_id", cmd.Flags().Lookup("issue-id"))
	viper.BindPFlag("state", cmd.Flags().Lookup("state"))
	viper.BindPFlag("labels", cmd.Flags().Lookup("labels"))
}

// Load reads the configuration from viper and returns a validated Config.
func Load() (*Config, error) {
	cfg := &Config{
		GitLabURL: viper.GetString("gitlab_url"),
		Token:     viper.GetString("token"),
		Project:   viper.GetString("project"),
		Output:    viper.GetString("output"),
	}

	if cfg.Token == "" {
		return nil, fmt.Errorf("gitlab token is required (use --token or GITLAB_TOKEN)")
	}
	if cfg.Project == "" {
		return nil, fmt.Errorf("project is required (use --project or GITLAB_PROJECT)")
	}

	return cfg, nil
}

// LoadIssues reads the full issues configuration.
func LoadIssues() (*IssuesConfig, error) {
	base, err := Load()
	if err != nil {
		return nil, err
	}

	return &IssuesConfig{
		Config:  *base,
		IssueID: viper.GetInt64("issue_id"),
		State:   viper.GetString("state"),
		Labels:  viper.GetStringSlice("labels"),
	}, nil
}
