package layout

import (
	"strings"

	"github.com/charmbracelet/x/ansi"
)

// ClampLines truncates each line to maxWidth visible characters and drops
// lines beyond maxLines. It avoids the Split → range → Join round-trip by
// scanning the source string directly with IndexByte (zero-allocation scan)
// and writing each accepted line to a single Builder.
//
// Allocations per call: 1 (Builder.Grow backing array) + 1 (result string).
func ClampLines(s string, maxWidth, maxLines int) string {
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return s
	}

	var b strings.Builder
	b.Grow(len(s)) // upper-bound pre-allocation; avoids mid-write resizes

	lineCount := 0
	for len(s) > 0 && lineCount < maxLines {
		idx := strings.IndexByte(s, '\n')
		var line string
		if idx < 0 {
			line = s
			s = ""
		} else {
			line = s[:idx]
			s = s[idx+1:]
		}

		if lineCount > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(ansi.Truncate(line, maxWidth, ""))
		lineCount++
	}

	return b.String()
}
