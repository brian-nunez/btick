package scheduler

import (
	"slices"
	"strings"
	"time"
)

func FormatTime(ts *time.Time) string {
	if ts == nil {
		return "-"
	}
	return ts.UTC().Format(time.RFC3339)
}

func FormatTimeValue(ts time.Time) string {
	if ts.IsZero() {
		return "-"
	}
	return ts.UTC().Format(time.RFC3339)
}

func IsRunSuccess(status string) bool {
	return strings.EqualFold(status, "success")
}

func IsRunFailure(status string) bool {
	return strings.EqualFold(status, "failed")
}

func HasScope(selected []string, scope string) bool {
	return slices.Contains(selected, scope)
}

func countEnabledJobs(jobs []UIJob) int {
	count := 0
	for _, job := range jobs {
		if job.Enabled {
			count++
		}
	}
	return count
}
