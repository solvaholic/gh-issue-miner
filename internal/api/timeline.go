package api

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// TimelineEvent represents a simplified timeline event relevant to references.
type TimelineEvent struct {
	ID                int64
	Type              string
	Actor             string
	CreatedAt         time.Time
	SourceOwnerRepo   string
	SourceIssueNumber int
}

// GetIssueTimeline fetches timeline events for an issue and returns events that reference other issues.
func GetIssueTimeline(ctx context.Context, client RESTClient, repo string, number int) ([]TimelineEvent, error) {
	var out []TimelineEvent
	page := 1
	perPage := 100
	for {
		path := fmt.Sprintf("repos/%s/issues/%d/timeline?per_page=%d&page=%d", repo, number, perPage, page)
		var raw interface{}
		if err := client.Get(path, &raw); err != nil {
			return nil, err
		}
		body, err := json.Marshal(raw)
		if err != nil {
			return nil, err
		}
		var items []map[string]interface{}
		if err := json.Unmarshal(body, &items); err != nil {
			return nil, err
		}
		if len(items) == 0 {
			break
		}
		for _, it := range items {
			var ev TimelineEvent
			if id, ok := it["id"].(float64); ok {
				ev.ID = int64(id)
			}
			// event type may be under "event" or "type"
			if t, ok := it["event"].(string); ok {
				ev.Type = t
			} else if t2, ok := it["type"].(string); ok {
				ev.Type = t2
			}
			if a, ok := it["actor"].(map[string]interface{}); ok {
				if login, ok := a["login"].(string); ok {
					ev.Actor = login
				}
			}
			if created, ok := it["created_at"].(string); ok {
				if tm, err := time.Parse(time.RFC3339, created); err == nil {
					ev.CreatedAt = tm
				}
			}

			// look for source.issue (cross-referenced)
			if src, ok := it["source"].(map[string]interface{}); ok {
				if iss, ok := src["issue"].(map[string]interface{}); ok {
					if num, ok := iss["number"].(float64); ok {
						ev.SourceIssueNumber = int(num)
					}
					// try to extract repository from issue.url or repository_url
					if u, ok := iss["url"].(string); ok {
						// url is like https://api.github.com/repos/owner/repo/issues/123
						// try to parse owner/repo
						// simple attempt: find "/repos/" and split
						idx := -1
						if p := "/repos/"; true {
							if i := indexOf(u, p); i >= 0 {
								idx = i + len(p)
							}
						}
						if idx >= 0 {
							tail := u[idx:]
							// tail like owner/repo/issues/123
							parts := splitPath(tail)
							if len(parts) >= 2 {
								ev.SourceOwnerRepo = parts[0] + "/" + parts[1]
							}
						}
					}
				}
			}

			// if event includes a direct "issue" field, it may be the source
			if iss, ok := it["issue"].(map[string]interface{}); ok {
				if num, ok := iss["number"].(float64); ok {
					ev.SourceIssueNumber = int(num)
				}
				if u, ok := iss["url"].(string); ok && ev.SourceOwnerRepo == "" {
					idx := indexOf(u, "/repos/")
					if idx >= 0 {
						tail := u[idx+len("/repos/"):]
						parts := splitPath(tail)
						if len(parts) >= 2 {
							ev.SourceOwnerRepo = parts[0] + "/" + parts[1]
						}
					}
				}
			}

			// only include events that reference another issue
			if ev.SourceIssueNumber != 0 {
				out = append(out, ev)
			}
		}
		if len(items) < perPage {
			break
		}
		page++
	}
	return out, nil
}

// helper: indexOf
func indexOf(s, sep string) int {
	return strings.Index(s, sep)
}

// helper: splitPath, split on '/' and remove empty
func splitPath(s string) []string {
	var parts []string
	for _, p := range strings.Split(s, "/") {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}
