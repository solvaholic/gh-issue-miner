package util

import "testing"

func TestParseIssueURL(t *testing.T) {
	tests := []struct {
		in       string
		wantRepo string
		wantNum  int
		ok       bool
	}{
		{"https://github.com/cli/cli/issues/12096", "cli/cli", 12096, true},
		{"http://github.com/owner/repo/issues/1", "owner/repo", 1, true},
		{"https://github.com/owner/repo/issues/", "", 0, false},
		{"not a url", "", 0, false},
		{"https://github.com/owner/repo/pull/5", "", 0, false},
	}

	for _, tt := range tests {
		r, n, ok := ParseIssueURL(tt.in)
		if ok != tt.ok {
			t.Fatalf("ParseIssueURL(%q) ok = %v, want %v", tt.in, ok, tt.ok)
		}
		if ok {
			if r != tt.wantRepo || n != tt.wantNum {
				t.Fatalf("ParseIssueURL(%q) = (%q,%d), want (%q,%d)", tt.in, r, n, tt.wantRepo, tt.wantNum)
			}
		}
	}
}
