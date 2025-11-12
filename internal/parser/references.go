package parser

import (
	"regexp"
	"strconv"
)

var (
	// full URL: https://github.com/owner/repo/issues/123
	reFullURL = regexp.MustCompile(`https?://[^/]+/([^/]+)/([^/]+)/issues/(\d+)`)
	// owner/repo#123
	reOwnerRef = regexp.MustCompile(`([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)#(\d+)`)
	// short reference: #123
	reShortRef = regexp.MustCompile(`#(\d+)`)
)

// Reference represents a detected issue reference.
type Reference struct {
	OwnerRepo string // empty when short ref
	Number    int
	Raw       string
}

// ParseReferences finds references in a text body. Returns unique references in order found.
func ParseReferences(s string) []Reference {
	var out []Reference
	seen := map[string]struct{}{}

	// full URLs
	for _, m := range reFullURL.FindAllStringSubmatch(s, -1) {
		if len(m) >= 4 {
			owner := m[1]
			repo := m[2]
			if n, err := strconv.Atoi(m[3]); err == nil {
				key := owner + "/" + repo + "#" + m[3]
				if _, ok := seen[key]; !ok {
					seen[key] = struct{}{}
					out = append(out, Reference{OwnerRepo: owner + "/" + repo, Number: n, Raw: m[0]})
				}
			}
		}
	}

	// owner/repo#123
	for _, m := range reOwnerRef.FindAllStringSubmatch(s, -1) {
		if len(m) >= 3 {
			if n, err := strconv.Atoi(m[2]); err == nil {
				key := m[1] + "#" + m[2]
				if _, ok := seen[key]; !ok {
					seen[key] = struct{}{}
					out = append(out, Reference{OwnerRepo: m[1], Number: n, Raw: m[0]})
				}
			}
		}
	}

	// short refs (#123) - these are ambiguous (same-repo) and should be included too
	for _, m := range reShortRef.FindAllStringSubmatch(s, -1) {
		if len(m) >= 2 {
			if n, err := strconv.Atoi(m[1]); err == nil {
				key := "#" + m[1]
				if _, ok := seen[key]; !ok {
					seen[key] = struct{}{}
					out = append(out, Reference{OwnerRepo: "", Number: n, Raw: m[0]})
				}
			}
		}
	}

	return out
}
