package docker

import (
	"errors"
	"testing"

	"github.com/moby/moby/api/types/container"
)

type mockStatusCodeError struct {
	code int
	msg  string
}

func (m mockStatusCodeError) Error() string { return m.msg }
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
