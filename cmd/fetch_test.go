package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

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
