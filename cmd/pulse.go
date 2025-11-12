package cmd

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
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
			issues = filterIssues(issues, pulseIncludePRs, pulseState, fallbackRaw)
		}

		metrics := analyzer.ComputePulse(issues)

		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintf(w, "Repository:\t%s\n\n", repoStr)

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
