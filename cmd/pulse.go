package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/solvaholic/gh-issue-miner/internal/analyzer"
	"github.com/solvaholic/gh-issue-miner/internal/api"
	"github.com/solvaholic/gh-issue-miner/internal/util"
)

var pulseRepo string
var pulseLimit int
var pulseIncludePRs bool
var pulseLabel string
var pulseState string
var pulseCreated string
var pulseUpdated string
var pulseClosed string

var pulseCmd = &cobra.Command{
	Use:   "pulse",
	Short: "Show metrics about repository issues",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := api.NewClient()
		if err != nil {
			return err
		}

		var issues []api.Issue
		var repoStr string

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
				if cmd.Flags().Changed("created") {
					conflict = append(conflict, "--created")
				}
				if cmd.Flags().Changed("updated") {
					conflict = append(conflict, "--updated")
				}
				if cmd.Flags().Changed("closed") {
					conflict = append(conflict, "--closed")
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

		if issues == nil {
			repo, err := util.DetectRepo(pulseRepo)
			if err != nil {
				return err
			}
			repoStr = repo

			// Expand label specs into exact labels for server-side querying
			labelsForAPI, fallbackRaw, err := ExpandLabelSpecs(ctx, client, repo, pulseLabel)
			if err != nil {
				return err
			}

			issues, err = api.ListIssues(ctx, client, repo, pulseLimit, pulseState, labelsForAPI, pulseIncludePRs)
			if err != nil {
				return err
			}

			// apply client-side filters only for any unmatched wildcard prefixes
			issues, err = filterIssues(issues, pulseIncludePRs, pulseState, fallbackRaw, pulseCreated, pulseUpdated, pulseClosed)
			if err != nil {
				return err
			}
		}

		metrics := analyzer.ComputePulse(issues)

		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintf(w, "Repository:\t%s\n\n", repoStr)

		// Print filter summary when non-default filters were provided
		var active []string
		if cmd.Flags().Changed("label") && pulseLabel != "" {
			active = append(active, fmt.Sprintf("label=%s", pulseLabel))
		}
		if cmd.Flags().Changed("state") && pulseState != "" {
			active = append(active, fmt.Sprintf("state=%s", pulseState))
		}
		if cmd.Flags().Changed("include-prs") && pulseIncludePRs {
			active = append(active, "include-prs=true")
		}
		if cmd.Flags().Changed("created") && pulseCreated != "" {
			active = append(active, fmt.Sprintf("created=%s", pulseCreated))
		}
		if cmd.Flags().Changed("updated") && pulseUpdated != "" {
			active = append(active, fmt.Sprintf("updated=%s", pulseUpdated))
		}
		if cmd.Flags().Changed("closed") && pulseClosed != "" {
			active = append(active, fmt.Sprintf("closed=%s", pulseClosed))
		}
		if cmd.Flags().Changed("limit") && pulseLimit != 100 {
			active = append(active, fmt.Sprintf("limit=%d", pulseLimit))
		}
		if len(active) > 0 {
			fmt.Fprintf(w, "Filters:\t%s\n\n", strings.Join(active, ", "))
		}

		twW, _, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil || twW <= 0 {
			twW = 80
		}

		maxVal := 0
		nums := []int{
			metrics.Open, metrics.Closed, metrics.Total,
			metrics.Opened7, metrics.Opened30, metrics.Opened90,
			metrics.Closed7, metrics.Closed30, metrics.Closed90,
		}
		for _, n := range nums {
			if n > maxVal {
				maxVal = n
			}
		}
		for _, v := range metrics.LabelCounts {
			if v > maxVal {
				maxVal = v
			}
		}
		for _, v := range metrics.AssigneeCounts {
			if v > maxVal {
				maxVal = v
			}
		}
		countWidth := 1
		if maxVal > 0 {
			countWidth = len(strconv.Itoa(maxVal))
		}

		fmt.Fprintln(w, "Issues:")
		fmt.Fprintf(w, "  Open:\t%*d\n  Closed:\t%*d\n  Total:\t%*d\n\n", countWidth, metrics.Open, countWidth, metrics.Closed, countWidth, metrics.Total)

		fmt.Fprintln(w, "Activity:")
		fmt.Fprintf(w, "  Opened (7d/30d/90d):\t%d / %d / %d\n", metrics.Opened7, metrics.Opened30, metrics.Opened90)
		fmt.Fprintf(w, "  Closed (7d/30d/90d):\t%d / %d / %d\n", metrics.Closed7, metrics.Closed30, metrics.Closed90)
		fmt.Fprintf(w, "  Avg time to close:\t%.1f days\n\n", metrics.AvgTimeToClose)

		fmt.Fprintln(w, "Most Active:")
		for _, it := range metrics.TopByComments {
			prefix := fmt.Sprintf("  #%d ", it.Number)
			suffix := fmt.Sprintf(" (%d comments)", it.Comments)
			avail := twW - len(prefix) - len(suffix) - 1
			if avail < 10 {
				avail = 30
			}
			title := truncateString(it.Title, avail)
			fmt.Fprintf(w, "%s%s%s\n", prefix, title, suffix)
		}
		fmt.Fprintln(w)

		type kv struct {
			K string
			V int
		}
		var labels []kv
		for k, v := range metrics.LabelCounts {
			labels = append(labels, kv{k, v})
		}
		sort.Slice(labels, func(i, j int) bool { return labels[i].V > labels[j].V })
		fmt.Fprintln(w, "Top Labels:")
		for i, it := range labels {
			if i >= 10 {
				break
			}
			fmt.Fprintf(w, "  %s\t%*d\n", it.K, countWidth, it.V)
		}
		fmt.Fprintln(w)

		fmt.Fprintln(w, "Assignees:")
		var ass []kv
		for k, v := range metrics.AssigneeCounts {
			ass = append(ass, kv{k, v})
		}
		sort.Slice(ass, func(i, j int) bool { return ass[i].V > ass[j].V })
		for i, it := range ass {
			if i >= 10 {
				break
			}
			fmt.Fprintf(w, "  %s\t%*d\n", it.K, countWidth, it.V)
		}

		w.Flush()

		return nil
	},
}

func init() {
	pulseCmd.Flags().StringVar(&pulseRepo, "repo", "", "Repository in owner/repo format (default: current repo)")
	pulseCmd.Flags().IntVar(&pulseLimit, "limit", 100, "Maximum number of issues to analyze")
	pulseCmd.Flags().BoolVar(&pulseIncludePRs, "include-prs", false, "Include pull requests in results")
	pulseCmd.Flags().StringVar(&pulseLabel, "label", "", "Comma-separated label specs (exact or prefix*). Matches issues containing any of these labels")
	pulseCmd.Flags().StringVar(&pulseState, "state", "", "Filter by issue state: open, closed")
	pulseCmd.Flags().StringVar(&pulseCreated, "created", "", "Filter by created timeframe (e.g., 7d, 2025-01-01, 2025-01-01..2025-01-31)")
	pulseCmd.Flags().StringVar(&pulseUpdated, "updated", "", "Filter by updated timeframe (e.g., 7d, 2025-01-01)")
	pulseCmd.Flags().StringVar(&pulseClosed, "closed", "", "Filter by closed timeframe (e.g., 30d, 2025-01-01..2025-02-01)")
	rootCmd.AddCommand(pulseCmd)
}

// truncateString truncates s to max runes and appends an ellipsis if truncated.
func truncateString(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 1 {
		return string(r[:max])
	}
	return string(r[:max-1]) + "…"
}
