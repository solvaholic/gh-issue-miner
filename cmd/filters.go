package cmd

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

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
// filterIssues applies include-prs, state, label, and time-range filters
// to the initial issue list. Time filters are provided as raw strings:
// - relative: `7d` means issues from now-7days..now
// - date: `2025-01-02` means that day
// - range: `2025-01-01..2025-01-31` inclusive
func filterIssues(issueList []api.Issue, includePRs bool, stateFilter string, labelRaw string, createdRaw string, updatedRaw string, closedRaw string) ([]api.Issue, error) {
	var out []api.Issue
	ls := parseLabelSpecs(labelRaw)
	// parse time ranges
	cStart, cEnd, cErr := parseTimeRange(createdRaw)
	if cErr != nil {
		return nil, cErr
	}
	uStart, uEnd, uErr := parseTimeRange(updatedRaw)
	if uErr != nil {
		return nil, uErr
	}
	clStart, clEnd, clErr := parseTimeRange(closedRaw)
	if clErr != nil {
		return nil, clErr
	}
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
		// created
		if createdRaw != "" {
			if !timeInRange(it.CreatedAt, cStart, cEnd) {
				continue
			}
		}
		// updated
		if updatedRaw != "" {
			if !timeInRange(it.UpdatedAt, uStart, uEnd) {
				continue
			}
		}
		// closed
		if closedRaw != "" {
			if it.ClosedAt == nil {
				continue
			}
			if !timeInRange(it.ClosedAt.UTC(), clStart, clEnd) {
				continue
			}
		}
		out = append(out, it)
	}
	return out, nil
}

// parseTimeRange parses supported time range syntaxes and returns start (inclusive)
// and end (exclusive) times in UTC. Empty input returns nil,nil,nil.
func parseTimeRange(raw string) (*time.Time, *time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil, nil
	}
	now := time.Now().UTC()
	// range like left..right where each side may be a date (YYYY-MM-DD)
	// or a relative like 7d. Handle ranges first so inputs like "60d..45d"
	// are parsed as intended.
	if strings.Contains(raw, "..") {
		parts := strings.SplitN(raw, "..", 2)
		a := strings.TrimSpace(parts[0])
		b := strings.TrimSpace(parts[1])
		var start *time.Time
		var end *time.Time

		// helper to parse a single side used for start (left) and end (right)
		parseSide := func(part string, isRight bool) (*time.Time, error) {
			if part == "" {
				return nil, nil
			}
			// relative like 7d
			if strings.HasSuffix(part, "d") {
				nstr := strings.TrimSuffix(part, "d")
				n, err := strconv.Atoi(nstr)
				if err != nil || n <= 0 {
					return nil, fmt.Errorf("invalid relative timeframe: %s", part)
				}
				tm := now.Add(time.Duration(-n) * 24 * time.Hour).UTC()
				// for range semantics we align to day boundaries: start -> 00:00, end -> 00:00 next day
				dayStart := time.Date(tm.Year(), tm.Month(), tm.Day(), 0, 0, 0, 0, time.UTC)
				if isRight {
					e := dayStart.Add(24 * time.Hour)
					return &e, nil
				}
				return &dayStart, nil
			}
			// ISO date YYYY-MM-DD
			if t, err := time.Parse("2006-01-02", part); err == nil {
				t = t.UTC()
				if isRight {
					e := t.Add(24 * time.Hour)
					return &e, nil
				}
				return &t, nil
			}
			return nil, fmt.Errorf("invalid date or relative timeframe: %s", part)
		}

		if a != "" {
			s, err := parseSide(a, false)
			if err != nil {
				return nil, nil, fmt.Errorf("invalid start date: %w", err)
			}
			start = s
		}
		if b != "" {
			e, err := parseSide(b, true)
			if err != nil {
				return nil, nil, fmt.Errorf("invalid end date: %w", err)
			}
			end = e
		}
		return start, end, nil
	}

	// relative like 7d (single relative window -> from now-N .. now)
	if strings.HasSuffix(raw, "d") {
		n := 0
		_, err := fmt.Sscanf(raw, "%dd", &n)
		if err != nil || n <= 0 {
			return nil, nil, fmt.Errorf("invalid relative timeframe: %s", raw)
		}
		start := now.Add(time.Duration(-n) * 24 * time.Hour)
		end := now.Add(time.Second)
		return &start, &end, nil
	}
	// single date YYYY-MM-DD
	if t, err := time.Parse("2006-01-02", raw); err == nil {
		start := t.UTC()
		end := start.Add(24 * time.Hour)
		return &start, &end, nil
	}
	return nil, nil, fmt.Errorf("unsupported time range format: %s", raw)
}

func timeInRange(t time.Time, start *time.Time, end *time.Time) bool {
	t = t.UTC()
	if start != nil && t.Before(*start) {
		return false
	}
	if end != nil && !t.Before(*end) {
		return false
	}
	return true
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
