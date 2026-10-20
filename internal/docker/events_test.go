package docker

import (
	"context"
	"reflect"
	"testing"

	"github.com/moby/moby/api/types/events"
)

func TestClient_StreamEvents(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name  string
		c     *Client
		args  args
		want  <-chan events.Message
		want1 <-chan error
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := tt.c.StreamEvents(tt.args.ctx)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Client.StreamEvents() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("Client.StreamEvents() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestIsContainerLifecycleEvent(t *testing.T) {
	type args struct {
		action events.Action
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsContainerLifecycleEvent(tt.args.action); got != tt.want {
				t.Errorf("IsContainerLifecycleEvent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsImageLifecycleEvent(t *testing.T) {
	type args struct {
		action events.Action
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsImageLifecycleEvent(tt.args.action); got != tt.want {
				t.Errorf("IsImageLifecycleEvent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsVolumeLifecycleEvent(t *testing.T) {
	type args struct {
		action events.Action
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsVolumeLifecycleEvent(tt.args.action); got != tt.want {
				t.Errorf("IsVolumeLifecycleEvent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsNetworkLifecycleEvent(t *testing.T) {
	type args struct {
		action events.Action
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNetworkLifecycleEvent(tt.args.action); got != tt.want {
				t.Errorf("IsNetworkLifecycleEvent() = %v, want %v", got, tt.want)
			}
		})
	}
}
