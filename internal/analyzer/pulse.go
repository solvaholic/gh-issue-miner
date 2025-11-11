package analyzer

import (
	"sort"
	"time"

	"github.com/solvaholic/gh-issue-miner/internal/api"
)

type PulseMetrics struct {
	Total          int
	Open           int
	Closed         int
	Opened7        int
	Opened30       int
	Opened90       int
	Closed7        int
	Closed30       int
	Closed90       int
	AvgTimeToClose float64 // days
	TopByComments  []api.Issue
	LabelCounts    map[string]int
	AssigneeCounts map[string]int
}

// ComputePulse computes basic metrics for the provided issues.
func ComputePulse(issues []api.Issue) PulseMetrics {
	now := time.Now()
	cutoff7 := now.AddDate(0, 0, -7)
	cutoff30 := now.AddDate(0, 0, -30)
	cutoff90 := now.AddDate(0, 0, -90)

	var pm PulseMetrics
	pm.LabelCounts = make(map[string]int)
	pm.AssigneeCounts = make(map[string]int)

	var totalCloseDuration time.Duration
	var closedCountForAvg int

	for _, it := range issues {
		pm.Total++
		if it.State == "open" {
			pm.Open++
		} else if it.State == "closed" {
			pm.Closed++
		}

		if it.CreatedAt.After(cutoff7) {
			pm.Opened7++
		}
		if it.CreatedAt.After(cutoff30) {
			pm.Opened30++
		}
		if it.CreatedAt.After(cutoff90) {
			pm.Opened90++
		}

		if it.ClosedAt != nil {
			if it.ClosedAt.After(cutoff7) {
				pm.Closed7++
			}
			if it.ClosedAt.After(cutoff30) {
				pm.Closed30++
			}
			if it.ClosedAt.After(cutoff90) {
				pm.Closed90++
			}
			// accumulate close durations
			dur := it.ClosedAt.Sub(it.CreatedAt)
			if dur > 0 {
				totalCloseDuration += dur
				closedCountForAvg++
			}
		}

		for _, l := range it.Labels {
			pm.LabelCounts[l]++
		}
		assignee := it.Assignee
		if assignee == "" {
			assignee = "unassigned"
		}
		pm.AssigneeCounts[assignee]++
	}

	if closedCountForAvg > 0 {
		avg := totalCloseDuration / time.Duration(closedCountForAvg)
		pm.AvgTimeToClose = avg.Hours() / 24.0
	}

	// top by comments
	sorted := make([]api.Issue, len(issues))
	copy(sorted, issues)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Comments > sorted[j].Comments
	})
	topN := 5
	if len(sorted) < topN {
		topN = len(sorted)
	}
	pm.TopByComments = sorted[:topN]

	return pm
}
