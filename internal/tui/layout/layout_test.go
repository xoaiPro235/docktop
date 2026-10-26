package layout

import "testing"

func TestClampLines(t *testing.T) {
	type args struct {
		s        string
		maxWidth int
		maxLines int
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "empty string",
			args: args{s: "", maxWidth: 10, maxLines: 5},
			want: "",
		},
		{
			name: "single line under maxWidth",
			args: args{s: "hello", maxWidth: 10, maxLines: 5},
			want: "hello",
		},
		{
			name: "single line exceeding maxWidth",
			args: args{s: "hello world!", maxWidth: 5, maxLines: 5},
			want: "hello",
		},
		{
			name: "multiple lines within maxLines",
			args: args{s: "line1\nline2\nline3", maxWidth: 10, maxLines: 5},
			want: "line1\nline2\nline3",
		},
		{
			name: "multiple lines exceeding maxLines",
			args: args{s: "line1\nline2\nline3\nline4\nline5", maxWidth: 10, maxLines: 2},
			want: "line1\nline2",
		},
		{
			name: "clamping width and lines together",
			args: args{s: "abcdefghij\n1234567890\nxyz", maxWidth: 4, maxLines: 2},
			want: "abcd\n1234",
		},
		{
			name: "trailing newlines are trimmed",
			args: args{s: "line1\n\n\n", maxWidth: 10, maxLines: 5},
			want: "line1",
		},
		{
			name: "maxLines is zero",
			args: args{s: "line1\nline2", maxWidth: 10, maxLines: 0},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ClampLines(tt.args.s, tt.args.maxWidth, tt.args.maxLines); got != tt.want {
				t.Errorf("ClampLines() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestListWindow(t *testing.T) {
	tests := []struct {
		name                  string
		height, cursor, total int
		hasFilter             bool
		wantStart, wantEnd    int
		wantMaxVisible        int
	}{
		{
			name:           "default window starts at first item",
			height:         10,
			cursor:         0,
			total:          20,
			wantStart:      0,
			wantEnd:        6,
			wantMaxVisible: 6,
		},
		{
			name:           "window follows cursor",
			height:         10,
			cursor:         9,
			total:          20,
			wantStart:      4,
			wantEnd:        10,
			wantMaxVisible: 6,
		},
		{
			name:           "filter reserves additional rows",
			height:         10,
			cursor:         5,
			total:          20,
			hasFilter:      true,
			wantStart:      2,
			wantEnd:        6,
			wantMaxVisible: 4,
		},
		{
			name:           "cramped height still shows one item",
			height:         1,
			cursor:         0,
			total:          2,
			wantStart:      0,
			wantEnd:        1,
			wantMaxVisible: 1,
		},
		{
			name:           "empty list",
			height:         10,
			cursor:         0,
			total:          0,
			wantStart:      0,
			wantEnd:        0,
			wantMaxVisible: 6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, maxVisible := ListWindow(tt.height, tt.cursor, tt.total, tt.hasFilter)
			if start != tt.wantStart || end != tt.wantEnd || maxVisible != tt.wantMaxVisible {
				t.Errorf("ListWindow() = (%d, %d, %d), want (%d, %d, %d)",
					start, end, maxVisible, tt.wantStart, tt.wantEnd, tt.wantMaxVisible)
			}
		})
	}
}

func TestListClickTarget(t *testing.T) {
	tests := []struct {
		name          string
		mouseY        int
		height        int
		cursor        int
		total         int
		hasFilter     bool
		wantIndex     int
		wantClickable bool
	}{
		{
			name:          "first item row",
			mouseY:        HeaderRowsDefault,
			height:        10,
			total:         10,
			wantIndex:     0,
			wantClickable: true,
		},
		{
			name:      "header is not clickable",
			mouseY:    HeaderRowsDefault - 1,
			height:    10,
			total:     10,
			wantIndex: -1,
		},
		{
			name:          "filtered list row uses search header offset",
			mouseY:        HeaderRowsWithSearch,
			height:        10,
			total:         10,
			hasFilter:     true,
			wantIndex:     0,
			wantClickable: true,
		},
		{
			name:          "click maps to scrolled item",
			mouseY:        HeaderRowsDefault + 2,
			height:        10,
			cursor:        8,
			total:         20,
			wantIndex:     5,
			wantClickable: true,
		},
		{
			name:      "click beyond visible items is not clickable",
			mouseY:    HeaderRowsDefault + 6,
			height:    10,
			total:     20,
			wantIndex: -1,
		},
		{
			name:      "empty list is not clickable",
			mouseY:    HeaderRowsDefault,
			height:    10,
			total:     0,
			wantIndex: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIndex, gotClickable := ListClickTarget(
				tt.mouseY, tt.height, tt.cursor, tt.total, tt.hasFilter,
			)
			if gotIndex != tt.wantIndex || gotClickable != tt.wantClickable {
				t.Errorf("ListClickTarget() = (%d, %v), want (%d, %v)",
					gotIndex, gotClickable, tt.wantIndex, tt.wantClickable)
			}
		})
	}
}
