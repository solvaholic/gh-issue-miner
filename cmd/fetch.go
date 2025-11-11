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

var fetchCmd = &cobra.Command{
	Use:   "fetch",
	Short: "Fetch list of issues from a repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		repo, err := util.DetectRepo(fetchRepo)
		if err != nil {
			return err
		}

		client, err := api.NewRESTClient()
		if err != nil {
			return err
		}

		issues, err := api.ListIssues(ctx, client, repo, fetchLimit)
		if err != nil {
			return err
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
}
