package format

import (
	"time"

	"github.com/docker/go-units"
)

// TimeAgo formats a time into a relative "X ago" string (e.g. "just now", "2 hours ago").
func TimeAgo(t time.Time) string {
	d := time.Since(t)
	if d < time.Minute {
		return "just now"
	}
	return units.HumanDuration(d) + " ago"
}

// Size formats a byte count into a human-readable size string (e.g. "150MB", "1.2GB").
func Size(bytes int64) string {
	return units.HumanSize(float64(bytes))
}
