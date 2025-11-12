package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

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

		// Ensure comments are fetched for every issue (list mode and single-issue mode)
		// We'll store comments per-issue so we can attribute references to comment authors/timestamps.
		issueComments := make(map[int][]api.Comment)
		var icMu sync.Mutex
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
				comments, _ := api.ListIssueComments(ctx, client, repo, it.Number)
				if len(comments) > 0 {
					icMu.Lock()
					issueComments[it.Number] = comments
					icMu.Unlock()
				}
			}(i)
		}
		wg.Wait()

		// Build adjacency with metadata. We'll use timeline events (if available) to annotate edges
		type Edge struct {
			Dest      string
			Actor     string
			Timestamp time.Time
			Action    string
			Source    string // "timeline", "comment", or "body"
			CommentID int64
		}

		// adj maps source issue -> map[dedupeKey]Edge to prevent duplicate edges
		adj := map[string]map[string]Edge{}

		// cache timeline lookups per destination to avoid repeated API calls
		timelineCache := map[string][]api.TimelineEvent{}
		var tcMu sync.Mutex

		// helper to get timeline events for a dest (owner/repo and number)
		getTimeline := func(ownerRepo string, number int) ([]api.TimelineEvent, error) {
			key := fmt.Sprintf("%s#%d", ownerRepo, number)
			tcMu.Lock()
			evs, ok := timelineCache[key]
			tcMu.Unlock()
			if ok {
				return evs, nil
			}
			evs, err := api.GetIssueTimeline(ctx, client, ownerRepo, number)
			if err != nil {
				return nil, err
			}
			tcMu.Lock()
			timelineCache[key] = evs
			tcMu.Unlock()
			return evs, nil
		}

		for _, it := range issues {
			// source repository for this issue (short refs resolve to this repo)
			srcRepo := repo
			srcKey := fmt.Sprintf("%s#%d", srcRepo, it.Number)

			// parse references in the issue body first
			bodyRefs := parser.ParseReferences(it.Body)
			for _, r := range bodyRefs {
				var destOwner string
				if r.OwnerRepo != "" {
					destOwner = r.OwnerRepo
				} else {
					// short refs (#123) resolve to the repository where they are mentioned
					destOwner = srcRepo
				}
				destKey := fmt.Sprintf("%s#%d", destOwner, r.Number)

				// attempt to annotate via timeline on the destination issue
				var edge Edge
				edge.Dest = destKey
				edge.Source = "body"

				if evs, err := getTimeline(destOwner, r.Number); err == nil {
					// look for an event where the source issue is this issue
					for _, ev := range evs {
						// match by source issue number and owner (owner may be empty if same repo)
						if ev.SourceIssueNumber == it.Number && (ev.SourceOwnerRepo == "" || ev.SourceOwnerRepo == srcRepo || ev.SourceOwnerRepo == destOwner) {
							edge.Actor = ev.Actor
							edge.Timestamp = ev.CreatedAt
							edge.Action = ev.Type
							edge.Source = "timeline"
							break
						}
					}
				}

				// dedupe key uses dest, source, actor, action, timestamp, comment id
				dk := fmt.Sprintf("%s|%s|%s|%s|%d|%d", edge.Dest, edge.Source, edge.Actor, edge.Action, edge.Timestamp.UnixNano(), edge.CommentID)
				if _, ok := adj[srcKey]; !ok {
					adj[srcKey] = map[string]Edge{}
				}
				adj[srcKey][dk] = edge
			}

			// parse references inside each comment and attribute to comment author/time
			if cms, ok := issueComments[it.Number]; ok {
				for _, c := range cms {
					crefs := parser.ParseReferences(c.Body)
					for _, r := range crefs {
						var destOwner string
						if r.OwnerRepo != "" {
							destOwner = r.OwnerRepo
						} else {
							// short refs in comments resolve to the repository where the comment was made
							destOwner = srcRepo
						}
						destKey := fmt.Sprintf("%s#%d", destOwner, r.Number)

						var edge Edge
						edge.Dest = destKey
						edge.Source = "comment"
						edge.Actor = c.Author
						edge.Timestamp = c.CreatedAt
						edge.CommentID = c.ID

						// prefer timeline metadata when available
						if evs, err := getTimeline(destOwner, r.Number); err == nil {
							for _, ev := range evs {
								if ev.SourceIssueNumber == it.Number && (ev.SourceOwnerRepo == "" || ev.SourceOwnerRepo == srcRepo || ev.SourceOwnerRepo == destOwner) {
									edge.Actor = ev.Actor
									edge.Timestamp = ev.CreatedAt
									edge.Action = ev.Type
									edge.Source = "timeline"
									edge.CommentID = 0
									break
								}
							}
						}

						dk := fmt.Sprintf("%s|%s|%s|%s|%d|%d", edge.Dest, edge.Source, edge.Actor, edge.Action, edge.Timestamp.UnixNano(), edge.CommentID)
						if _, ok := adj[srcKey]; !ok {
							adj[srcKey] = map[string]Edge{}
						}
						adj[srcKey][dk] = edge
					}
				}
			}
		}

		// Print adjacency list with metadata
		for src, edges := range adj {
			fmt.Fprintf(os.Stdout, "%s\n", src)
			for _, e := range edges {
				var meta []string
				meta = append(meta, fmt.Sprintf("source=%s", e.Source))
				if e.Actor != "" {
					meta = append(meta, fmt.Sprintf("actor=%s", e.Actor))
				}
				if !e.Timestamp.IsZero() {
					meta = append(meta, fmt.Sprintf("at=%s", e.Timestamp.Format(time.RFC3339)))
				}
				if e.Action != "" {
					meta = append(meta, fmt.Sprintf("action=%s", e.Action))
				}
				if e.CommentID != 0 {
					meta = append(meta, fmt.Sprintf("comment_id=%d", e.CommentID))
				}
				fmt.Fprintf(os.Stdout, "  -> %s  (%s)\n", e.Dest, strings.Join(meta, ", "))
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
