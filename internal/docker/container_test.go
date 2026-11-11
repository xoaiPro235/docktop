package docker

import (
	"errors"
	"os/exec"
	"reflect"
	"testing"

	"github.com/moby/moby/api/types/container"
)

type mockStatusCodeError struct {
	code int
	msg  string
}

func (m mockStatusCodeError) Error() string   { return m.msg }
func (m mockStatusCodeError) StatusCode() int { return m.code }

func Test_isConflictError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"status code 409 error", mockStatusCodeError{code: 409, msg: "conflict"}, true},
		{"status code 500 error", mockStatusCodeError{code: 500, msg: "server error"}, false},
		{"container is running string", errors.New("container is running, cannot remove"), true},
		{"conflict string", errors.New("conflict: name already in use"), true},
		{"unrelated error", errors.New("connection reset by peer"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isConflictError(tt.err); got != tt.want {
				t.Errorf("isConflictError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShortID(t *testing.T) {
	tests := []struct {
		name string
		id   string
		want string
	}{
		{"long hash 64 chars", "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890", "abcdef123456"},
		{"exact 12 chars", "123456789012", "123456789012"},
		{"short id 8 chars", "12345678", "12345678"},
		{"empty string", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShortID(tt.id); got != tt.want {
				t.Errorf("ShortID(%q) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}

func TestContainerInfo_ShortID(t *testing.T) {
	c := &ContainerInfo{ID: "fedcba9876543210"}
	if got := c.ShortID(); got != "fedcba987654" {
		t.Errorf("ContainerInfo.ShortID() = %v, want fedcba987654", got)
	}
}

func TestContainerInfo_HasDetailsLoaded(t *testing.T) {
	c1 := &ContainerInfo{Details: nil}
	if c1.HasDetailsLoaded() {
		t.Error("HasDetailsLoaded() should be false when Details is nil")
	}

	c2 := &ContainerInfo{Details: &container.InspectResponse{}}
	if !c2.HasDetailsLoaded() {
		t.Error("HasDetailsLoaded() should be true when Details is non-nil")
	}
}

func TestClient_ExecShellCmd(t *testing.T) {
	tests := []struct {
		name string
		host string
		want []string
	}{
		{
			name: "configured host",
			host: "ssh://docker@example.com",
			want: []string{"docker", "-H", "ssh://docker@example.com", "exec", "-it", "container-id", "sh", "-c", "command -v bash >/dev/null 2>&1 && exec bash || exec sh"},
		},
		{
			name: "default host",
			want: []string{"docker", "exec", "-it", "container-id", "sh", "-c", "command -v bash >/dev/null 2>&1 && exec bash || exec sh"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := (&Client{host: tt.host}).ExecShellCmd("container-id").Args
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ExecShellCmd().Args = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestWebURL(t *testing.T) {
	tests := []struct {
		name     string
		ports    string
		expected string
		ok       bool
	}{
		{
			name:     "standard 0.0.0.0 mapping",
			ports:    "0.0.0.0:8080->80/tcp",
			expected: "http://localhost:8080/",
			ok:       true,
		},
		{
			name:     "custom IP mapping",
			ports:    "127.0.0.1:3000->3000/tcp",
			expected: "http://127.0.0.1:3000/",
			ok:       true,
		},
		{
			name:     "ipv6 mapping",
			ports:    "[::]:8080->80/tcp",
			expected: "http://localhost:8080/",
			ok:       true,
		},
		{
			name:     "short host port format",
			ports:    "8080->80/tcp",
			expected: "http://localhost:8080/",
			ok:       true,
		},
		{
			name:     "internal port only without mapping",
			ports:    "80/tcp",
			expected: "",
			ok:       false,
		},
		{
			name:     "multiple ports picks first mapped",
			ports:    "80/tcp, 0.0.0.0:8080->80/tcp, 0.0.0.0:9000->9000/tcp",
			expected: "http://localhost:8080/",
			ok:       true,
		},
		{
			name:     "empty ports",
			ports:    "",
			expected: "",
			ok:       false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			info := ContainerInfo{Ports: tc.ports}
			got, ok := info.WebURL()
			if ok != tc.ok {
				t.Fatalf("expected ok=%v, got %v", tc.ok, ok)
			}
			if got != tc.expected {
				t.Errorf("expected URL %q, got %q", tc.expected, got)
			}
		})
	}
}

func TestOpenBrowser(t *testing.T) {
	orig := openBrowserCmd
	defer func() { openBrowserCmd = orig }()

	var capturedURL string
	openBrowserCmd = func(url string) *exec.Cmd {
		capturedURL = url
		return exec.Command("true")
	}

	testURL := "http://localhost:8080/"
	if err := OpenBrowser(testURL); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedURL != testURL {
		t.Errorf("expected capturedURL %q, got %q", testURL, capturedURL)
	}
}
