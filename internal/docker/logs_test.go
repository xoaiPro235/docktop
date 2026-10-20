package docker

import (
	"context"
	"io"
	"reflect"
	"testing"
)

func TestClient_StreamContainerLogs(t *testing.T) {
	type args struct {
		ctx  context.Context
		id   string
		tail string
	}
	tests := []struct {
		name string
		c    *Client
		args args
		want <-chan string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.StreamContainerLogs(tt.args.ctx, tt.args.id, tt.args.tail); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Client.StreamContainerLogs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_StreamComposeLogs(t *testing.T) {
	type args struct {
		ctx  context.Context
		opts ComposeLogOpts
	}
	tests := []struct {
		name string
		c    *Client
		args args
		want <-chan string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.c.StreamComposeLogs(tt.args.ctx, tt.args.opts); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Client.StreamComposeLogs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_scanLogs(t *testing.T) {
	type args struct {
		ctx context.Context
		r   io.Reader
		out chan<- string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scanLogs(tt.args.ctx, tt.args.r, tt.args.out)
		})
	}
}
