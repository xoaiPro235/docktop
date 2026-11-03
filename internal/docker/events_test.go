package docker

import (
	"testing"

	"github.com/moby/moby/api/types/events"
)

func TestIsContainerLifecycleEvent(t *testing.T) {
	tests := []struct {
		name   string
		action events.Action
		want   bool
	}{
		{name: "start", action: events.ActionStart, want: true},
		{name: "destroy", action: events.ActionDestroy, want: true},
		{name: "health status is not a lifecycle event", action: events.ActionHealthStatus, want: false},
		{name: "unknown action", action: events.Action("unknown"), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsContainerLifecycleEvent(tt.action); got != tt.want {
				t.Errorf("IsContainerLifecycleEvent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsImageLifecycleEvent(t *testing.T) {
	tests := []struct {
		name   string
		action events.Action
		want   bool
	}{
		{name: "pull", action: events.ActionPull, want: true},
		{name: "untag", action: events.ActionUnTag, want: true},
		{name: "container start is not an image event", action: events.ActionStart, want: false},
		{name: "unknown action", action: events.Action("unknown"), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsImageLifecycleEvent(tt.action); got != tt.want {
				t.Errorf("IsImageLifecycleEvent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsVolumeLifecycleEvent(t *testing.T) {
	tests := []struct {
		name   string
		action events.Action
		want   bool
	}{
		{name: "mount", action: events.ActionMount, want: true},
		{name: "unmount", action: events.ActionUnmount, want: true},
		{name: "image pull is not a volume event", action: events.ActionPull, want: false},
		{name: "unknown action", action: events.Action("unknown"), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsVolumeLifecycleEvent(tt.action); got != tt.want {
				t.Errorf("IsVolumeLifecycleEvent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsNetworkLifecycleEvent(t *testing.T) {
	tests := []struct {
		name   string
		action events.Action
		want   bool
	}{
		{name: "connect", action: events.ActionConnect, want: true},
		{name: "remove", action: events.ActionRemove, want: true},
		{name: "container stop is not a network event", action: events.ActionStop, want: false},
		{name: "unknown action", action: events.Action("unknown"), want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNetworkLifecycleEvent(tt.action); got != tt.want {
				t.Errorf("IsNetworkLifecycleEvent() = %v, want %v", got, tt.want)
			}
		})
	}
}
