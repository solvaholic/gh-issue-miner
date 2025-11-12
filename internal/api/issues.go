package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// Issue is a minimal issue representation used for output and analysis.
type Issue struct {
	Number    int
	State     string
	Title     string
	Body      string
	Labels    []string
	Assignee  string
	CreatedAt time.Time
	UpdatedAt time.Time
	ClosedAt  *time.Time
	Comments  int
}

// ListIssues lists issues for the given repo (owner/repo) up to limit.
func ListIssues(ctx context.Context, client RESTClient, repo string, limit int) ([]Issue, error) {
	var result []Issue
	if limit <= 0 {
		limit = 100
	}

	perPage := 100
	page := 1

	for len(result) < limit {
		// build path
		qs := url.Values{}
		qs.Set("state", "all")
		qs.Set("per_page", strconv.Itoa(perPage))
		qs.Set("page", strconv.Itoa(page))
		path := fmt.Sprintf("repos/%s/issues?%s", repo, qs.Encode())

		// Use client.Get into an interface{}, then marshal/unmarshal to work with dynamic JSON
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
			if len(result) >= limit {
				break
			}
			var iss Issue
			if n, ok := it["number"].(float64); ok {
				iss.Number = int(n)
			}
			if s, ok := it["state"].(string); ok {
				iss.State = s
			}
			if t, ok := it["title"].(string); ok {
				iss.Title = t
			}
			if b, ok := it["body"].(string); ok {
				iss.Body = b
			}
			if comments, ok := it["comments"].(float64); ok {
				iss.Comments = int(comments)
			}
			if created, ok := it["created_at"].(string); ok {
				if tm, err := time.Parse(time.RFC3339, created); err == nil {
					iss.CreatedAt = tm
				}
			}
			if updated, ok := it["updated_at"].(string); ok {
				if tm, err := time.Parse(time.RFC3339, updated); err == nil {
					iss.UpdatedAt = tm
				}
			}
			if closed, ok := it["closed_at"].(string); ok && closed != "" {
				if tm, err := time.Parse(time.RFC3339, closed); err == nil {
					iss.ClosedAt = &tm
				}
			}
			// labels
			if lbls, ok := it["labels"].([]interface{}); ok {
				for _, l := range lbls {
					if lm, ok := l.(map[string]interface{}); ok {
						if name, ok := lm["name"].(string); ok {
							iss.Labels = append(iss.Labels, name)
						}
					}
				}
			}
			// assignee
			if asg, ok := it["assignee"].(map[string]interface{}); ok && asg != nil {
				if login, ok := asg["login"].(string); ok {
					iss.Assignee = login
				}
			}

			result = append(result, iss)
		}

		if len(items) < perPage {
			break
		}
		page++
	}

	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

// GetIssue fetches a single issue by number from the given repo (owner/repo).
func GetIssue(ctx context.Context, client RESTClient, repo string, number int) (Issue, error) {
	var iss Issue
	path := fmt.Sprintf("repos/%s/issues/%d", repo, number)
	var raw interface{}
	if err := client.Get(path, &raw); err != nil {
		return iss, err
	}
	body, err := json.Marshal(raw)
	if err != nil {
		return iss, err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(body, &m); err != nil {
		return iss, err
	}
	if n, ok := m["number"].(float64); ok {
		iss.Number = int(n)
	}
	if s, ok := m["state"].(string); ok {
		iss.State = s
	}
	if t, ok := m["title"].(string); ok {
		iss.Title = t
	}
	if b, ok := m["body"].(string); ok {
		iss.Body = b
	}
	if comments, ok := m["comments"].(float64); ok {
		iss.Comments = int(comments)
	}
	if created, ok := m["created_at"].(string); ok {
		if tm, err := time.Parse(time.RFC3339, created); err == nil {
			iss.CreatedAt = tm
		}
	}
	if updated, ok := m["updated_at"].(string); ok {
		if tm, err := time.Parse(time.RFC3339, updated); err == nil {
			iss.UpdatedAt = tm
		}
	}
	if closed, ok := m["closed_at"].(string); ok && closed != "" {
		if tm, err := time.Parse(time.RFC3339, closed); err == nil {
			iss.ClosedAt = &tm
		}
	}
	if lbls, ok := m["labels"].([]interface{}); ok {
		for _, l := range lbls {
			if lm, ok := l.(map[string]interface{}); ok {
				if name, ok := lm["name"].(string); ok {
					iss.Labels = append(iss.Labels, name)
				}
			}
		}
	}
	if asg, ok := m["assignee"].(map[string]interface{}); ok && asg != nil {
		if login, ok := asg["login"].(string); ok {
			iss.Assignee = login
		}
	}
	return iss, nil
}
