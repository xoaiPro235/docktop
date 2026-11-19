package format

import (
	"strings"
	"testing"
	"time"
)

func TestTimeAgo(t *testing.T) {
	now := time.Now()

	if got := TimeAgo(now.Add(-10 * time.Second)); got != "just now" {
		t.Errorf("TimeAgo(10s ago) = %q, want 'just now'", got)
	}

	twoHoursAgo := now.Add(-2 * time.Hour)
	gotTwoHours := TimeAgo(twoHoursAgo)
	if !strings.HasSuffix(gotTwoHours, "ago") {
		t.Errorf("TimeAgo(2h ago) = %q, expected to end with 'ago'", gotTwoHours)
	}
}

func TestSize(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0B"},
		{1024, "1kB"},
		{1024 * 1024 * 150, "157.3MB"},
	}

	for _, tt := range tests {
		got := Size(tt.bytes)
		if len(got) == 0 {
			t.Errorf("Size(%d) returned empty string", tt.bytes)
		}
	}
}
