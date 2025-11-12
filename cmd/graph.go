package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/spf13/cobra"

	"github.com/solvaholic/gh-issue-miner/internal/api"
	"github.com/solvaholic/gh-issue-miner/internal/parser"
	"github.com/solvaholic/gh-issue-miner/internal/util"
)

var graphRepo string
var graphLimit int

var graphCmd = &cobra.Command{
	Use:   "graph",
	Short: "Build a relationship graph from issues",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		client, err := api.NewClient()
		if err != nil {
			return err
		}

		var issues []api.Issue
		repo := graphRepo

		// If given a positional issue URL, fetch that issue and its comments
		if len(args) > 0 {
			if r, num, ok := util.ParseIssueURL(args[0]); ok {
				repo = r
				single, err := api.GetIssue(ctx, client, r, num)
				if err != nil {
					return err
				}
				issues = []api.Issue{single}
				// fetch comments too for references
				comments, _ := api.ListIssueComments(ctx, client, r, num)
				// attach comments into the parsing step below by creating a pseudo-issue
				// we'll represent comments as additional bodies to parse for this single issue
				if len(comments) > 0 {
					// create a dummy issue that concatenates issue body and comment bodies
					var sb strings.Builder
					sb.WriteString(single.Body)
					for _, c := range comments {
						sb.WriteString("\n\n")
						sb.WriteString(c.Body)
					}
					// replace the single issue body with concatenated text for parsing
					issues[0].Body = sb.String()
				}
			}
		}

		if issues == nil {
			if repo == "" {
				r, err := util.DetectRepo(graphRepo)
				if err != nil {
					return err
				}
				repo = r
			}
			issues, err = api.ListIssues(ctx, client, repo, graphLimit)
			if err != nil {
				return err
			}
		}

		// Ensure comments are parsed for every issue (list mode and single-issue mode)
		// Fetch comments concurrently with a small worker pool to avoid bursting the API
		sem := make(chan struct{}, 5)
		var wg sync.WaitGroup
		for i := range issues {
			wg.Add(1)
			sem <- struct{}{}
			go func(idx int) {
				defer wg.Done()
				defer func() { <-sem }()
				it := &issues[idx]
				// fetch comments for this issue; ignore errors but include any bodies we get
				comments, _ := api.ListIssueComments(ctx, client, repo, it.Number)
				if len(comments) > 0 {
					var sb strings.Builder
					sb.WriteString(it.Body)
					for _, c := range comments {
						sb.WriteString("\n\n")
						sb.WriteString(c.Body)
					}
					it.Body = sb.String()
				}
			}(i)
		}
		wg.Wait()

		// Build a simple adjacency map by parsing bodies for references
		adj := map[string][]string{}
		for _, it := range issues {
			key := fmt.Sprintf("%s#%d", repo, it.Number)
			refs := parser.ParseReferences(it.Body)
			for _, r := range refs {
				var dest string
				if r.OwnerRepo != "" {
					dest = fmt.Sprintf("%s#%d", r.OwnerRepo, r.Number)
				} else {
					dest = fmt.Sprintf("%s#%d", repo, r.Number)
				}
				adj[key] = append(adj[key], dest)
			}
		}

		// Print adjacency list
		for src, dsts := range adj {
			fmt.Fprintf(os.Stdout, "%s\n", src)
			for _, d := range dsts {
				fmt.Fprintf(os.Stdout, "  -> %s\n", d)
			}
		}
		return nil
	},
}

func init() {
	graphCmd.Flags().StringVar(&graphRepo, "repo", "", "Repository in owner/repo format (default: current repo)")
	graphCmd.Flags().IntVar(&graphLimit, "limit", 100, "Maximum number of issues to include in the graph")
	rootCmd.AddCommand(graphCmd)
}
