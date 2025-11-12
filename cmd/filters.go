package cmd

import (
	"context"
	"strings"

	"github.com/solvaholic/gh-issue-miner/internal/api"
)

// labelSpec holds compiled label matching rules.
type labelSpec struct {
	exact    map[string]bool
	prefixes []string
}

// parseLabelSpecs takes a comma-separated list of label specs like
// "initiative,epic,batch,foo-*" and returns a labelSpec for matching.
func parseLabelSpecs(raw string) labelSpec {
	s := labelSpec{exact: map[string]bool{}}
	if raw == "" {
		return s
	}
	for _, part := range strings.Split(raw, ",") {
		p := strings.TrimSpace(part)
		if p == "" {
			continue
		}
		if strings.HasSuffix(p, "*") {
			s.prefixes = append(s.prefixes, strings.TrimSuffix(p, "*"))
		} else {
			s.exact[p] = true
		}
	}
	return s
}

// matches returns true if any label matches the spec
func (ls *labelSpec) matches(labels []string) bool {
	if ls == nil {
		return false
	}
	for _, l := range labels {
		if ls.exact[l] {
			return true
		}
		for _, p := range ls.prefixes {
			if strings.HasPrefix(l, p) {
				return true
			}
		}
	}
	return false
}

// filterIssues applies include-prs, state, and label filtering to the initial issue list.
func filterIssues(issueList []api.Issue, includePRs bool, stateFilter string, labelRaw string) []api.Issue {
	var out []api.Issue
	ls := parseLabelSpecs(labelRaw)
	// allow empty stateFilter to mean no filtering
	for _, it := range issueList {
		if !includePRs && it.IsPR {
			continue
		}
		if stateFilter != "" {
			if !strings.EqualFold(it.State, stateFilter) {
				continue
			}
		}
		if labelRaw != "" {
			if !ls.matches(it.Labels) {
				continue
			}
		}
		out = append(out, it)
	}
	return out
}

// ExpandLabelSpecs expands comma-separated label specs where trailing '*' indicates
// a prefix wildcard. It returns a list of exact labels suitable for server-side
// querying and a fallbackRaw string (comma-separated) containing any wildcard
// specs that did not match any repo label (to be applied client-side).
func ExpandLabelSpecs(ctx context.Context, client api.RESTClient, repo string, raw string) ([]string, string, error) {
	var exact []string
	var fallback []string
	if raw == "" {
		return exact, "", nil
	}

	// detect whether we need to fetch repo labels
	needLabels := false
	parts := strings.Split(raw, ",")
	for _, p := range parts {
		if strings.HasSuffix(strings.TrimSpace(p), "*") {
			needLabels = true
			break
		}
	}

	var repoLabels []string
	if needLabels {
		rl, err := api.ListRepoLabels(ctx, client, repo)
		if err != nil {
			// return error so callers can decide; do not silently ignore
			return nil, "", err
		}
		repoLabels = rl
	}

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if strings.HasSuffix(p, "*") {
			prefix := strings.TrimSuffix(p, "*")
			found := false
			for _, rl := range repoLabels {
				if strings.HasPrefix(rl, prefix) {
					exact = append(exact, rl)
					found = true
				}
			}
			if !found {
				fallback = append(fallback, p)
			}
			continue
		}
		exact = append(exact, p)
	}

	return exact, strings.Join(fallback, ","), nil
}
