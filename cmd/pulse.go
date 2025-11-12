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

var pulseCmd = &cobra.Command{
	Use:   "pulse",
	Short: "Show metrics about repository issues",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := api.NewClient()
		if err != nil {
			return err
		}

		// If a positional arg is provided and it's an issue URL, handle single-issue mode.
		var issues []api.Issue
		var repoStr string
		if len(args) > 0 {
			if r, num, ok := util.ParseIssueURL(args[0]); ok {
				// fetched single issue
				single, err := api.GetIssue(ctx, client, r, num)
				if err != nil {
					return err
				}
				issues = []api.Issue{single}
				repoStr = r
			}
		}

		// otherwise behave as before (repo detection + list)
		if issues == nil {
			repo, err := util.DetectRepo(pulseRepo)
			if err != nil {
				return err
			}
			repoStr = repo
			issues, err = api.ListIssues(ctx, client, repo, pulseLimit)
			if err != nil {
				return err
			}
		}

		metrics := analyzer.ComputePulse(issues)

		// Use tabwriter for neat alignment
		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintf(w, "Repository:\t%s\n\n", repoStr)

		// Determine terminal width for title truncation
		tw, _, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil || tw <= 0 {
			tw = 80
		}

		// compute max width among counts for right-justification
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
			avail := tw - len(prefix) - len(suffix) - 1
			if avail < 10 {
				avail = 30
			}
			title := truncateString(it.Title, avail)
			fmt.Fprintf(w, "%s%s%s\n", prefix, title, suffix)
		}
		fmt.Fprintln(w)

		// Top labels
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
