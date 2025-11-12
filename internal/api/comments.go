package api

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Comment represents a minimal issue comment.
type Comment struct {
	Author    string
	Body      string
	CreatedAt time.Time
}

// ListIssueComments fetches comments for an issue (paginated).
func ListIssueComments(ctx context.Context, client RESTClient, repo string, number int) ([]Comment, error) {
	var out []Comment
	page := 1
	perPage := 100
	for {
		path := fmt.Sprintf("repos/%s/issues/%d/comments?per_page=%d&page=%d", repo, number, perPage, page)
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
			var c Comment
			if b, ok := it["body"].(string); ok {
				c.Body = b
			}
			if a, ok := it["user"].(map[string]interface{}); ok {
				if login, ok := a["login"].(string); ok {
					c.Author = login
				}
			}
			if created, ok := it["created_at"].(string); ok {
				if tm, err := time.Parse(time.RFC3339, created); err == nil {
					c.CreatedAt = tm
				}
			}
			out = append(out, c)
		}
		if len(items) < perPage {
			break
		}
		page++
	}
	return out, nil
}
