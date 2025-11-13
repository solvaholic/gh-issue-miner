package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/solvaholic/gh-issue-miner/internal/api"
	"github.com/solvaholic/gh-issue-miner/internal/util"
)

var fetchRepo string
var fetchLimit int
var fetchIncludePRs bool
var fetchLabel string
var fetchState string
var fetchCreated string
var fetchUpdated string
var fetchClosed string
var fetchSort string
var fetchDirection string

var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch list of issues from a repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		client, err := api.NewClient()
		if err != nil {
			return err
		}

		var issues []api.Issue
		var repoStr string

		// If a positional arg is provided and it's an issue URL, treat it as exclusive with filters.
		if len(args) > 0 {
			if r, num, ok := util.ParseIssueURL(args[0]); ok {
				var conflict []string
				if cmd.Flags().Changed("label") {
					conflict = append(conflict, "--label")
				}
				if cmd.Flags().Changed("state") {
					conflict = append(conflict, "--state")
				}
				if cmd.Flags().Changed("include-prs") {
					conflict = append(conflict, "--include-prs")
				}
				if cmd.Flags().Changed("repo") {
					conflict = append(conflict, "--repo")
				}
				if cmd.Flags().Changed("limit") {
					conflict = append(conflict, "--limit")
				}
				if len(conflict) > 0 {
					return fmt.Errorf("positional issue URL cannot be combined with filters: %s", strings.Join(conflict, ", "))
				}

				single, err := api.GetIssue(ctx, client, r, num)
				if err != nil {
					return err
				}
				issues = []api.Issue{single}
				repoStr = r
			}
		}

		// Otherwise, list issues from detected repo
		if issues == nil {
			repo, err := util.DetectRepo(fetchRepo)
			if err != nil {
				return err
			}
			repoStr = repo

			// validate sort/direction
			if fetchSort != "" {
				switch fetchSort {
				case "created", "updated", "comments":
				default:
					return fmt.Errorf("invalid --sort value: %s (allowed: created, updated, comments)", fetchSort)
				}
			}
			if fetchDirection != "" {
				d := strings.ToLower(fetchDirection)
				if d != "asc" && d != "desc" {
					return fmt.Errorf("invalid --direction/--order value: %s (allowed: asc, desc)", fetchDirection)
				}
			}

			// Expand label specs into exact labels for server-side querying
			labelsForAPI, fallbackRaw, err := ExpandLabelSpecs(ctx, client, repo, fetchLabel)
			if err != nil {
				return err
			}

			// determine candidate limit to allow client-side filtering without prematurely truncating
			candidateLimit := fetchLimit
			if candidateLimit > 0 {
				// fetch a bit more candidates to account for client-side filtering
				candidateLimit = candidateLimit * 3
				const maxCandidates = 2000
				if candidateLimit > maxCandidates {
					candidateLimit = maxCandidates
				}
			}

			// if updated filter has a start bound, push it to the server via `since`
			uStart, _, uErr := parseTimeRange(fetchUpdated)
			if uErr != nil {
				return uErr
			}

			issues, err = api.ListIssues(ctx, client, repo, candidateLimit, fetchState, labelsForAPI, fetchIncludePRs, fetchSort, strings.ToLower(fetchDirection), uStart)
			if err != nil {
				return err
			}

			// Apply client-side filters only for any unmatched wildcard prefixes
			issues, err = filterIssues(issues, fetchIncludePRs, fetchState, fallbackRaw, fetchCreated, fetchUpdated, fetchClosed)
			if err != nil {
				return err
			}

			// Trim to requested limit after client-side filtering
			if fetchLimit > 0 && len(issues) > fetchLimit {
				issues = issues[:fetchLimit]
			}
		}

		// Print repo header when available
		if repoStr != "" {
			fmt.Fprintf(os.Stdout, "Repository:\t%s\n\n", repoStr)
		}

		// use tabwriter for aligned columns
		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(w, "#\tstate\ttitle\tlabels\tassignee\tcreated\tupdated\tcomments")
		for _, it := range issues {
			labels := strings.Join(it.Labels, ",")
			assignee := "unassigned"
			if it.Assignee != "" {
				assignee = it.Assignee
			}
			title := it.Title
			// Quote title to keep spaces visible
			title = fmt.Sprintf("\"%s\"", title)
			fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%s\t%d\n",
				it.Number,
				it.State,
				title,
				labels,
				assignee,
				it.CreatedAt.Format("2006-01-02"),
				it.UpdatedAt.Format("2006-01-02"),
				it.Comments,
			)
		}
		w.Flush()

		return nil
	},
}

func init() {
	fetchCmd.Flags().StringVar(&fetchRepo, "repo", "", "Repository in owner/repo format (default: current repo)")
	fetchCmd.Flags().IntVar(&fetchLimit, "limit", 100, "Maximum number of issues to fetch")
	fetchCmd.Flags().BoolVar(&fetchIncludePRs, "include-prs", false, "Include pull requests in results")
	fetchCmd.Flags().StringVar(&fetchLabel, "label", "", "Comma-separated label specs (exact or prefix*). Matches issues containing any of these labels")
	fetchCmd.Flags().StringVar(&fetchState, "state", "", "Filter by issue state: open, closed")
	fetchCmd.Flags().StringVar(&fetchCreated, "created", "", "Filter by created timeframe (e.g., 7d, 2025-01-01, 2025-01-01..2025-01-31)")
	fetchCmd.Flags().StringVar(&fetchUpdated, "updated", "", "Filter by updated timeframe (e.g., 7d, 2025-01-01)")
	fetchCmd.Flags().StringVar(&fetchClosed, "closed", "", "Filter by closed timeframe (e.g., 30d, 2025-01-01..2025-02-01)")
	fetchCmd.Flags().StringVar(&fetchSort, "sort", "", "Sort field: created, updated, comments")
	fetchCmd.Flags().StringVar(&fetchDirection, "direction", "", "Sort direction: asc or desc")
	// alias --order to --direction for discoverability (bind to same variable)
	fetchCmd.Flags().StringVar(&fetchDirection, "order", "", "Alias for --direction")
}
