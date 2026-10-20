package docker

import (
	"strings"
	"testing"
	"time"

	"github.com/moby/moby/api/types/image"
)

func TestCleanLayerCommand(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		want string
	}{
		{
			name: "dockerfile nop instruction",
			cmd:  "/bin/sh -c #(nop)  ENV PATH=/usr/local/bin:$PATH",
			want: "ENV PATH=/usr/local/bin:$PATH",
		},
		{
			name: "shell command without RUN prefix",
			cmd:  "/bin/sh -c apt-get update && apt-get install -y curl",
			want: "RUN apt-get update && apt-get install -y curl",
		},
		{
			name: "shell command with explicit RUN prefix",
			cmd:  "/bin/sh -c RUN make build",
			want: "RUN make build",
		},
		{
			name: "empty command",
			cmd:  "   ",
			want: "<missing>",
		},
		{
			name: "standard command line",
			cmd:  "COPY . /app",
			want: "COPY . /app",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CleanLayerCommand(tt.cmd); got != tt.want {
				t.Errorf("CleanLayerCommand(%q) = %q, want %q", tt.cmd, got, tt.want)
			}
		})
	}
}

func TestFormatTimeAgo(t *testing.T) {
	now := time.Now()
	if got := FormatTimeAgo(now.Add(-10 * time.Second)); got != "just now" {
		t.Errorf("FormatTimeAgo(10s ago) = %q, want 'just now'", got)
	}

	twoHoursAgo := now.Add(-2 * time.Hour)
	gotTwoHours := FormatTimeAgo(twoHoursAgo)
	if !strings.Contains(gotTwoHours, "ago") {
		t.Errorf("FormatTimeAgo(2h ago) = %q, expected to end with 'ago'", gotTwoHours)
	}
}

func TestFormatImageSize(t *testing.T) {
	size := int64(1024 * 1024 * 50) // ~50MB
	got := FormatImageSize(size)
	if !strings.Contains(got, "MB") && !strings.Contains(got, "B") {
		t.Errorf("FormatImageSize(%d) = %q", size, got)
	}
}

func TestCleanImageID(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want string
	}{
		{"with sha256 prefix", "sha256:719f31ab9820bcde", "719f31ab98"},
		{"without sha256 prefix", "719f31ab9820bcde", "719f31ab98"},
		{"short id", "sha256:abc", "abc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CleanImageID(tt.id); got != tt.want {
				t.Errorf("CleanImageID(%q) = %q, want %q", tt.id, got, tt.want)
			}
		})
	}
}

func Test_displayTag(t *testing.T) {
	tests := []struct {
		name         string
		raw          image.Summary
		wantTag      string
		wantDangling bool
	}{
		{
			name: "named repo tag",
			raw: image.Summary{
				RepoTags: []string{"postgres:16-alpine"},
			},
			wantTag:      "postgres:16-alpine",
			wantDangling: false,
		},
		{
			name: "digest fallback",
			raw: image.Summary{
				RepoTags:    []string{"<none>:<none>"},
				RepoDigests: []string{"docker.io/library/nginx@sha256:abcdef123456"},
			},
			wantTag:      "docker.io/library/nginx:<digest>",
			wantDangling: false,
		},
		{
			name: "dangling untagged image",
			raw: image.Summary{
				RepoTags:    []string{"<none>:<none>"},
				RepoDigests: []string{"<none>@<none>"},
			},
			wantTag:      "<none>:<none>",
			wantDangling: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotTag, gotDangling := displayTag(tt.raw)
			if gotTag != tt.wantTag || gotDangling != tt.wantDangling {
				t.Errorf("displayTag() = (%q, %v), want (%q, %v)", gotTag, gotDangling, tt.wantTag, tt.wantDangling)
			}
		})
	}
}
