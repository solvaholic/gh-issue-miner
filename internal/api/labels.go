package api

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// ListRepoLabels returns the list of label names configured on the repository.
func ListRepoLabels(ctx context.Context, client RESTClient, repo string) ([]string, error) {
	var out []string
	page := 1
	perPage := 100
	for {
		path := fmt.Sprintf("repos/%s/labels?per_page=%d&page=%d", repo, perPage, page)
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
			if n, ok := it["name"].(string); ok {
				out = append(out, n)
			}
		}
		if len(items) < perPage {
			break
		}
		page++
		// small throttle in case callers loop rapidly
		time.Sleep(10 * time.Millisecond)
	}
	return out, nil
}
