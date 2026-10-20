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
