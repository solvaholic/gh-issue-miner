package cmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/solvaholic/gh-issue-miner/internal/api"
)

// fakeClient implements api.RESTClient for tests.
type fakeClient struct{}

func (f *fakeClient) Get(path string, out interface{}) error {
	// If path is a single-issue GET, return a map representing the issue
	if strings.HasPrefix(path, "repos/cli/cli/issues/") && !strings.Contains(path, "?") {
		m := map[string]interface{}{
			"number":     12096,
			"state":      "open",
			"title":      "Example issue title",
			"comments":   3,
			"created_at": "2025-01-01T12:00:00Z",
			"updated_at": "2025-01-02T12:00:00Z",
			"labels":     []interface{}{map[string]interface{}{"name": "bug"}},
			"assignee":   map[string]interface{}{"login": "alice"},
		}
		if ptr, ok := out.(*interface{}); ok {
			*ptr = m
			return nil
		}
		return nil
	}

	// For list endpoints, return an array with one item
	if strings.HasPrefix(path, "repos/cli/cli/issues") {
		items := []map[string]interface{}{
			{
				"number":     12096,
				"state":      "open",
				"title":      "Example issue title",
				"comments":   3,
				"created_at": "2025-01-01T12:00:00Z",
				"updated_at": "2025-01-02T12:00:00Z",
				"labels":     []interface{}{map[string]interface{}{"name": "bug"}},
				"assignee":   map[string]interface{}{"login": "alice"},
			},
		}
		if ptr, ok := out.(*interface{}); ok {
			*ptr = items
			return nil
		}
		return nil
	}

	return nil
}

func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	outC := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		outC <- buf.String()
	}()

	f()
	_ = w.Close()
	os.Stdout = old
	return <-outC
}

func TestFetchWithIssueURL(t *testing.T) {
	// replace api.NewRESTClient with a fake that returns our fake client
	orig := api.NewClient
	api.NewClient = func() (api.RESTClient, error) {
		return &fakeClient{}, nil
	}
	defer func() { api.NewClient = orig }()

	out := captureOutput(func() {
		err := fetchCmd.RunE(fetchCmd, []string{"https://github.com/cli/cli/issues/12096"})
		if err != nil {
			t.Fatalf("fetch run error: %v", err)
		}
	})

	if !strings.Contains(out, "12096") || !strings.Contains(out, "Example issue title") {
		t.Fatalf("unexpected fetch output: %s", out)
	}
}

func TestFetchIssues_WithMockList(t *testing.T) {
	old := api.ListIssuesFunc
	defer func() { api.ListIssuesFunc = old }()

	// Prepare mock issues (5 issues with increasing numbers)
	now := time.Now().UTC()
	issuesAll := []api.Issue{}
	for i := 1; i <= 5; i++ {
		issuesAll = append(issuesAll, api.Issue{
			Number:    i,
			State:     "open",
			Title:     "issue",
			CreatedAt: now.AddDate(0, 0, -i),
			UpdatedAt: now.AddDate(0, 0, -i),
			Comments:  i,
		})
	}

	// Mock ListIssuesFunc to return all issues regardless of the limit passed
	api.ListIssuesFunc = func(ctx context.Context, client api.RESTClient, repo string, limit int, state string, labels []string, includePRs bool, sort string, direction string, since *time.Time) ([]api.Issue, error) {
		// Verify that since is nil for this call
		if since != nil {
			return nil, errors.New("unexpected since in mock")
		}
		return issuesAll, nil
	}

	// Call FetchIssues asking for limit=2, expect to get first 2 after client-side filtering (none here)
	out, repo, err := FetchIssues(context.Background(), (api.RESTClient)(nil), "owner/repo", 2, false, "", "", "", "", "", "", "")
	if err != nil {
		t.Fatalf("FetchIssues returned error: %v", err)
	}
	if repo != "owner/repo" {
		t.Fatalf("expected repo owner/repo, got %s", repo)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 results, got %d", len(out))
	}
	if out[0].Number != 1 || out[1].Number != 2 {
		t.Fatalf("unexpected items returned: %v", out)
	}
}

func TestFetchIssues_UpdatedSincePassed(t *testing.T) {
	old := api.ListIssuesFunc
	defer func() { api.ListIssuesFunc = old }()

	// Mock ListIssuesFunc to capture since param
	var capturedSince *time.Time
	api.ListIssuesFunc = func(ctx context.Context, client api.RESTClient, repo string, limit int, state string, labels []string, includePRs bool, sort string, direction string, since *time.Time) ([]api.Issue, error) {
		capturedSince = since
		return []api.Issue{}, nil
	}

	// Call with updated range that has a start bound
	_, _, err := FetchIssues(context.Background(), (api.RESTClient)(nil), "owner/repo", 10, false, "", "", "", "60d..45d", "", "", "")
	if err != nil {
		t.Fatalf("FetchIssues returned error: %v", err)
	}
	if capturedSince == nil {
		t.Fatalf("expected since to be passed to ListIssuesFunc")
	}
}
