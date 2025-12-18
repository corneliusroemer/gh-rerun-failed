package main

import (
	"fmt"
	"os"
	"time"

	"github.com/cli/gh-rerun-failed/internal/gh"
	"github.com/cli/gh-rerun-failed/internal/rerunner"
	"github.com/spf13/cobra"
)

var (
	repoOverride  string
	branch        string
	limit         int
	sinceStr      string
	prNumber      int
	allOpenPRs    bool
		dryRun           bool
		failedOnly       bool
		includeDrafts    bool
		includeCancelled bool
		includeTimedOut  bool
	)
	
	func main() {
		rootCmd := &cobra.Command{
			Use:   "gh-rerun-failed",
			Short: "Rerun failed GitHub Actions runs with ease",
			Long:  `A GitHub CLI extension to rerun failed workflow runs across branches, commits, and PRs.`,
			RunE: func(cmd *cobra.Command, args []string) error {
				return runRerunner()
			},
		}
	
		rootCmd.Flags().StringVarP(&repoOverride, "repo", "R", "", "Select another repository using the [HOST/]OWNER/REPO format")
		rootCmd.Flags().StringVarP(&branch, "branch", "b", "", "Filter runs by branch")
		rootCmd.Flags().IntVarP(&limit, "limit", "L", 0, "Limit the number of runs to process")
		rootCmd.Flags().StringVarP(&sinceStr, "since", "s", "", "Only process runs since this duration (e.g. 24h, 1h)")
		rootCmd.Flags().IntVar(&prNumber, "pr", 0, "Filter runs by PR number")
		rootCmd.Flags().BoolVar(&allOpenPRs, "all-prs", false, "Process runs for all open PRs")
		rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be done without performing re-runs")
		rootCmd.Flags().BoolVar(&failedOnly, "failed-only", true, "Only rerun failed jobs within a run")
		rootCmd.Flags().BoolVar(&includeDrafts, "include-drafts", false, "Include draft PRs when using --all-prs")
		rootCmd.Flags().BoolVar(&includeCancelled, "include-cancelled", false, "Include cancelled runs")
		rootCmd.Flags().BoolVar(&includeTimedOut, "include-timed-out", false, "Include timed-out runs")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runRerunner() error {
	var since time.Duration
	if sinceStr != "" {
		var err error
		since, err = time.ParseDuration(sinceStr)
		if err != nil {
			return fmt.Errorf("invalid duration for --since: %w", err)
		}
	}

	client, err := gh.NewClient(repoOverride)
	if err != nil {
		return err
	}

	opts := rerunner.Options{
		Repo:          repoOverride,
		Branch:        branch,
		Limit:         limit,
		Since:         since,
		PRNumber:      prNumber,
		AllOpenPRs:    allOpenPRs,
		DryRun:           dryRun,
		FailedOnly:       failedOnly,
		IncludeDrafts:    includeDrafts,
		IncludeCancelled: includeCancelled,
		IncludeTimedOut:  includeTimedOut,
	}

	r := rerunner.NewRerunner(client, opts)
	return r.Run()
}