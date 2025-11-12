package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/solvaholic/gh-issue-miner/internal/api"
)

// fakeRESTClient implements api.RESTClient for tests by returning canned responses
type fakeRESTClient struct {
	responses map[string]interface{}
}

func (f *fakeRESTClient) Get(path string, v interface{}) error {
	// find best-match key (exact) or prefix match ignoring query
	p := path
	// strip query string
	if i := strings.Index(p, "?"); i >= 0 {
		p = p[:i]
	}
	resp, ok := f.responses[p]
	if !ok {
		// try prefix match
		for k, val := range f.responses {
			if strings.HasPrefix(p, k) {
				resp = val
				ok = true
				break
			}
		}
	}
	if !ok {
		return fmt.Errorf("no fake response for %s", path)
	}
	// marshal then unmarshal into v to mimic behavior of real client
	b, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

func TestGraphTraversalAndCycleAndCrossRepo(t *testing.T) {
	// prepare fake data
	now := time.Now().UTC().Format(time.RFC3339)
	// repos: r1/r1 has issue 1 referencing #2 and owner2/repo#3
	// r1/r1 issue 2 references #1 (cycle)
	// owner2/repo issue 3 has no refs

	r := map[string]interface{}{
		"repos/r1/r1/issues/1": map[string]interface{}{
			"number":   1,
			"body":     "See #2 and owner2/repo#3",
			"comments": 1,
		},
		"repos/r1/r1/issues/2": map[string]interface{}{
			"number":   2,
			"body":     "Backref to #1",
			"comments": 0,
		},
		"repos/owner2/repo/issues/3": map[string]interface{}{
			"number":   3,
			"body":     "External issue",
			"comments": 0,
		},
		"repos/r1/r1/issues/1/comments": []map[string]interface{}{
			{
				"id":         101,
				"body":       "comment with #2",
				"user":       map[string]interface{}{"login": "alice"},
				"created_at": now,
			},
		},
		"repos/r1/r1/issues/1/timeline":       []interface{}{},
		"repos/r1/r1/issues/2/timeline":       []interface{}{},
		"repos/owner2/repo/issues/3/timeline": []interface{}{},
	}

	fake := &fakeRESTClient{responses: r}

	oldNew := api.NewClient
	api.NewClient = func() (api.RESTClient, error) { return fake, nil }
	defer func() { api.NewClient = oldNew }()

	// capture stdout
	oldOut := os.Stdout
	rReader, rWriter, _ := os.Pipe()
	os.Stdout = rWriter

	// run graph for single issue r1/r1#1 with depth=1 and cross-repo=false
	graphDepth = 1
	graphCrossRepo = false
	if err := graphCmd.RunE(graphCmd, []string{"https://github.com/r1/r1/issues/1"}); err != nil {
		t.Fatalf("graph run failed: %v", err)
	}

	// restore stdout and read
	rWriter.Close()
	var buf bytes.Buffer
	buf.ReadFrom(rReader)
	os.Stdout = oldOut
	out := buf.String()

	// expect r1/r1 header, r1/r1#1 -> r1/r1#2 and owner2/repo#3
	if !strings.Contains(out, "r1/r1#1") {
		t.Fatalf("output missing source r1/r1#1: %s", out)
	}
	if !strings.Contains(out, "-> r1/r1#2") {
		t.Fatalf("output missing edge to r1/r1#2: %s", out)
	}
	if !strings.Contains(out, "-> owner2/repo#3") {
		t.Fatalf("output missing edge to owner2/repo#3: %s", out)
	}
	// since cross-repo is false, owner2/repo#3 should not appear as a source header
	if strings.Contains(out, "owner2/repo#3\n") {
		t.Fatalf("owner2/repo#3 was expanded but cross-repo is false: %s", out)
	}

	// ensure cycle handled: r1/r1#2 should be present and show edge back to r1/r1#1
	if !strings.Contains(out, "r1/r1#2") || !strings.Contains(out, "-> r1/r1#1") {
		t.Fatalf("cycle edges missing: %s", out)
	}

	// Now test depth=2 and cross-repo=true to expand external repo
	// capture stdout again
	oldOut = os.Stdout
	rReader, rWriter, _ = os.Pipe()
	os.Stdout = rWriter

	graphDepth = 2
	graphCrossRepo = true
	if err := graphCmd.RunE(graphCmd, []string{"https://github.com/r1/r1/issues/1"}); err != nil {
		t.Fatalf("graph run failed: %v", err)
	}

	rWriter.Close()
	buf.Reset()
	buf.ReadFrom(rReader)
	os.Stdout = oldOut
	out2 := buf.String()

	// expect owner2/repo#3 to appear as a source header now
	if !strings.Contains(out2, "owner2/repo#3#3") && !strings.Contains(out2, "owner2/repo#3\n") {
		// accept either header format; just check that owner2/repo#3 appears as a source
		t.Fatalf("expected owner2/repo#3 to be expanded in output: %s", out2)
	}
}
