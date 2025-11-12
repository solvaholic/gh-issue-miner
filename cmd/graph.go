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
var graphDepth int
var graphCrossRepo bool
var graphMaxNodes int
var graphIncludePRs bool
var graphLabel string
var graphState string
var graphCreated string
var graphUpdated string
var graphClosed string

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
		var fallbackPrefixes []string

		// If given a positional issue URL, fetch that issue and its comments
		if len(args) > 0 {
			if r, num, ok := util.ParseIssueURL(args[0]); ok {
				// positional URL is exclusive with selection filters
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
				if cmd.Flags().Changed("created") {
					conflict = append(conflict, "--created")
				}
				if cmd.Flags().Changed("updated") {
					conflict = append(conflict, "--updated")
				}
				if cmd.Flags().Changed("closed") {
					conflict = append(conflict, "--closed")
				}
				if len(conflict) > 0 {
					return fmt.Errorf("positional issue URL cannot be combined with filters: %s", strings.Join(conflict, ", "))
				}

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
			// Prepare labels for server-side filtering: expand trailing-* patterns
			var labelsForAPI []string
			var fallbackPrefixes []string
			if graphLabel != "" {
				needLabels := false
				for _, p := range strings.Split(graphLabel, ",") {
					if strings.HasSuffix(strings.TrimSpace(p), "*") {
						needLabels = true
						break
					}
				}

				var repoLabels []string
				if needLabels {
					repoLabels, _ = api.ListRepoLabels(ctx, client, repo)
				}

				for _, p := range strings.Split(graphLabel, ",") {
					p = strings.TrimSpace(p)
					if p == "" {
						continue
					}
					if strings.HasSuffix(p, "*") {
						prefix := strings.TrimSuffix(p, "*")
						found := false
						for _, rl := range repoLabels {
							if strings.HasPrefix(rl, prefix) {
								labelsForAPI = append(labelsForAPI, rl)
								found = true
							}
						}
						if !found {
							fallbackPrefixes = append(fallbackPrefixes, p)
						}
						continue
					}
					labelsForAPI = append(labelsForAPI, p)
				}
			}

			issues, err = api.ListIssues(ctx, client, repo, graphLimit, graphState, labelsForAPI, graphIncludePRs)
			if err != nil {
				return err
			}
		}

		// apply filters only to the initial issue selection
		// But if we expanded wildcard patterns above, apply fallback client-side filtering
		fallbackRaw := strings.Join(fallbackPrefixes, ",")
		if fallbackRaw != "" {
			issues, err = filterIssues(issues, graphIncludePRs, graphState, fallbackRaw, graphCreated, graphUpdated, graphClosed)
			if err != nil {
				return err
			}
		} else {
			issues, err = filterIssues(issues, graphIncludePRs, graphState, graphLabel, graphCreated, graphUpdated, graphClosed)
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

		// Use a semaphore to limit concurrent timeline fetches and an inflight map
		timelineSem := make(chan struct{}, 5)
		type tlResult struct {
			evs []api.TimelineEvent
			err error
		}
		inflight := map[string]chan tlResult{}
		var inflightMu sync.Mutex

		// helper to get timeline events for a dest (owner/repo and number)
		getTimeline := func(ownerRepo string, number int) ([]api.TimelineEvent, error) {
			key := fmt.Sprintf("%s#%d", ownerRepo, number)
			// check cache
			tcMu.Lock()
			evs, ok := timelineCache[key]
			tcMu.Unlock()
			if ok {
				return evs, nil
			}

			// dedupe inflight requests
			inflightMu.Lock()
			ch, ok := inflight[key]
			if ok {
				inflightMu.Unlock()
				res := <-ch
				return res.evs, res.err
			}
			ch = make(chan tlResult, 1)
			inflight[key] = ch
			inflightMu.Unlock()

			// perform fetch in a goroutine but block here until it's done
			go func() {
				timelineSem <- struct{}{}
				evs, err := api.GetIssueTimeline(ctx, client, ownerRepo, number)
				<-timelineSem

				if err == nil {
					tcMu.Lock()
					timelineCache[key] = evs
					tcMu.Unlock()
				}

				inflightMu.Lock()
				ch <- tlResult{evs: evs, err: err}
				delete(inflight, key)
				inflightMu.Unlock()
				close(ch)
			}()

			res := <-ch
			return res.evs, res.err
		}

		// We'll perform a breadth-first traversal up to graphDepth, starting from the initial issues.
		type visitItem struct {
			Repo   string
			Number int
			Depth  int
		}

		maxDepth := graphDepth
		allowCross := graphCrossRepo

		// cache fetched issues by key owner/repo#num
		issuesCache := map[string]api.Issue{}
		// comments by key
		commentsCache := map[string][]api.Comment{}

		// seeded queue: initial issues
		var q []visitItem
		nodesSeen := map[string]bool{}
		nodesCount := 0
		limitHit := false
		for _, it := range issues {
			key := fmt.Sprintf("%s#%d", repo, it.Number)
			issuesCache[key] = it
			if !nodesSeen[key] {
				if graphMaxNodes > 0 && nodesCount >= graphMaxNodes {
					limitHit = true
				} else {
					nodesSeen[key] = true
					nodesCount++
					q = append(q, visitItem{Repo: repo, Number: it.Number, Depth: 0})
				}
			}
		}

		// visited set for cycle detection
		visited := map[string]bool{}

		for len(q) > 0 {
			cur := q[0]
			q = q[1:]
			srcKey := fmt.Sprintf("%s#%d", cur.Repo, cur.Number)
			if visited[srcKey] {
				continue
			}
			visited[srcKey] = true

			// ensure a header is present even if this node has no outgoing edges
			if _, ok := adj[srcKey]; !ok {
				adj[srcKey] = map[string]Edge{}
			}

			// ensure issue is fetched
			it, ok := issuesCache[srcKey]
			if !ok {
				fetched, err := api.GetIssue(ctx, client, cur.Repo, cur.Number)
				if err != nil {
					// skip if we cannot fetch the issue
					continue
				}
				it = fetched
				issuesCache[srcKey] = it
			}

			// fetch comments if not present
			if _, ok := commentsCache[srcKey]; !ok {
				cms, _ := api.ListIssueComments(ctx, client, cur.Repo, cur.Number)
				if len(cms) > 0 {
					commentsCache[srcKey] = cms
				}
			}

			// parse refs from body
			srcRepo := cur.Repo
			bodyRefs := parser.ParseReferences(it.Body)
			for _, r := range bodyRefs {
				var destOwner string
				if r.OwnerRepo != "" {
					destOwner = r.OwnerRepo
				} else {
					destOwner = srcRepo
				}
				destKey := fmt.Sprintf("%s#%d", destOwner, r.Number)

				var edge Edge
				edge.Dest = destKey
				edge.Source = "body"

				if evs, err := getTimeline(destOwner, r.Number); err == nil {
					for _, ev := range evs {
						if ev.SourceIssueNumber == cur.Number && (ev.SourceOwnerRepo == "" || ev.SourceOwnerRepo == srcRepo || ev.SourceOwnerRepo == destOwner) {
							edge.Actor = ev.Actor
							edge.Timestamp = ev.CreatedAt
							edge.Action = ev.Type
							edge.Source = "timeline"
							break
						}
					}
				}

				dk := fmt.Sprintf("%s|%s|%s|%s|%d|%d", edge.Dest, edge.Source, edge.Actor, edge.Action, edge.Timestamp.UnixNano(), edge.CommentID)
				if _, ok := adj[srcKey]; !ok {
					adj[srcKey] = map[string]Edge{}
				}
				adj[srcKey][dk] = edge

				// follow this destination if depth allows
				if cur.Depth+1 <= maxDepth {
					// decide cross-repo expansion
					if destOwner == cur.Repo || allowCross {
						// enqueue dest if we haven't seen it and haven't hit the node limit
						if !visited[destKey] {
							if !nodesSeen[destKey] {
								if graphMaxNodes > 0 && nodesCount >= graphMaxNodes {
									limitHit = true
								} else {
									nodesSeen[destKey] = true
									nodesCount++
									q = append(q, visitItem{Repo: destOwner, Number: r.Number, Depth: cur.Depth + 1})
								}
							}
						}
					}
				}
			}

			// parse refs from comments and attribute
			if cms, ok := commentsCache[srcKey]; ok {
				for _, c := range cms {
					crefs := parser.ParseReferences(c.Body)
					for _, r := range crefs {
						var destOwner string
						if r.OwnerRepo != "" {
							destOwner = r.OwnerRepo
						} else {
							destOwner = srcRepo
						}
						destKey := fmt.Sprintf("%s#%d", destOwner, r.Number)

						var edge Edge
						edge.Dest = destKey
						edge.Source = "comment"
						edge.Actor = c.Author
						edge.Timestamp = c.CreatedAt
						edge.CommentID = c.ID

						if evs, err := getTimeline(destOwner, r.Number); err == nil {
							for _, ev := range evs {
								if ev.SourceIssueNumber == cur.Number && (ev.SourceOwnerRepo == "" || ev.SourceOwnerRepo == srcRepo || ev.SourceOwnerRepo == destOwner) {
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

						if cur.Depth+1 <= maxDepth {
							if destOwner == cur.Repo || allowCross {
								if !visited[destKey] {
									if !nodesSeen[destKey] {
										if graphMaxNodes > 0 && nodesCount >= graphMaxNodes {
											limitHit = true
										} else {
											nodesSeen[destKey] = true
											nodesCount++
											q = append(q, visitItem{Repo: destOwner, Number: r.Number, Depth: cur.Depth + 1})
										}
									}
								}
							}
						}
					}
				}

				if limitHit {
					fmt.Fprintf(os.Stderr, "warning: traversal hit --max-nodes=%d; some referenced nodes were not expanded\n", graphMaxNodes)
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
	graphCmd.Flags().IntVar(&graphDepth, "depth", 1, "Traversal depth for following references (default: 1)")
	graphCmd.Flags().BoolVar(&graphCrossRepo, "cross-repo", false, "Allow following references across repositories when recursing")
	graphCmd.Flags().IntVar(&graphMaxNodes, "max-nodes", 500, "Maximum number of nodes to visit during traversal (0 = unlimited)")
	graphCmd.Flags().BoolVar(&graphIncludePRs, "include-prs", false, "Include pull requests in the initial issue selection")
	graphCmd.Flags().StringVar(&graphLabel, "label", "", "Comma-separated label specs (exact or prefix*). Matches issues containing any of these labels")
	graphCmd.Flags().StringVar(&graphState, "state", "", "Filter by issue state: open, closed")
	graphCmd.Flags().StringVar(&graphCreated, "created", "", "Filter by created timeframe (e.g., 7d, 2025-01-01, 2025-01-01..2025-01-31)")
	graphCmd.Flags().StringVar(&graphUpdated, "updated", "", "Filter by updated timeframe (e.g., 7d, 2025-01-01)")
	graphCmd.Flags().StringVar(&graphClosed, "closed", "", "Filter by closed timeframe (e.g., 30d, 2025-01-01..2025-02-01)")
	rootCmd.AddCommand(graphCmd)
}
