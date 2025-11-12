package util

import (
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
)

var issueURLRe = regexp.MustCompile(`(?i)^https?://[^/]+/([^/]+)/([^/]+)/issues/(\d+)$`)

// ParseIssueURL attempts to parse a GitHub issue URL and returns owner/repo and issue number.
// Returns repo as "owner/repo" and issue number (int) on success, otherwise empty/0 and false.
func ParseIssueURL(s string) (string, int, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", 0, false
	}
	// quick regex match
	if m := issueURLRe.FindStringSubmatch(s); len(m) == 4 {
		num, err := strconv.Atoi(m[3])
		if err != nil {
			return "", 0, false
		}
		repo := m[1] + "/" + m[2]
		return repo, num, true
	}

	// try url parsing and tolerant patterns
	u, err := url.Parse(s)
	if err != nil {
		return "", 0, false
	}
	// path should be /owner/repo/issues/number (possibly with trailing .git removed)
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) >= 4 && parts[len(parts)-2] == "issues" {
		owner := parts[len(parts)-4+0]
		repo := parts[len(parts)-4+1]
		// last part is number
		numPart := parts[len(parts)-1]
		if n, err := strconv.Atoi(numPart); err == nil {
			return owner + "/" + strings.TrimSuffix(repo, ".git"), n, true
		}
	}

	// last attempt: split path and look for "issues" segment
	p := path.Clean(u.Path)
	segs := strings.Split(strings.Trim(p, "/"), "/")
	for i := range segs {
		if segs[i] == "issues" && i >= 2 && i+1 < len(segs) {
			owner := segs[i-2]
			repo := segs[i-1]
			if n, err := strconv.Atoi(segs[i+1]); err == nil {
				return owner + "/" + strings.TrimSuffix(repo, ".git"), n, true
			}
		}
	}

	return "", 0, false
}
