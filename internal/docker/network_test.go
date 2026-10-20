package docker

import (
	"context"
	"reflect"
	"testing"

	"github.com/moby/moby/api/types/network"
)

func TestClient_ListNetworks(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		c       *Client
		args    args
		want    []NetworkInfo
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.c.ListNetworks(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.ListNetworks() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Client.ListNetworks() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_networkScore(t *testing.T) {
	tests := []struct {
		name string
		n    NetworkInfo
		want int
	}{
		{
			name: "system network gets score 3",
			n:    NetworkInfo{IsSystem: true, ConnectedCount: 5},
			want: 3,
		},
		{
			name: "user network with connected containers gets score 1",
			n:    NetworkInfo{IsSystem: false, ConnectedCount: 2},
			want: 1,
		},
		{
			name: "empty user network gets score 2",
			n:    NetworkInfo{IsSystem: false, ConnectedCount: 0},
			want: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := networkScore(tt.n); got != tt.want {
				t.Errorf("networkScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_InspectNetwork(t *testing.T) {
	type args struct {
		ctx       context.Context
		networkID string
	}
	tests := []struct {
		name    string
		c       *Client
		args    args
		want    *network.Inspect
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.c.InspectNetwork(tt.args.ctx, tt.args.networkID)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.InspectNetwork() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Client.InspectNetwork() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_RemoveNetwork(t *testing.T) {
	type args struct {
		ctx       context.Context
		networkID string
	}
	tests := []struct {
		name    string
		c       *Client
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.c.RemoveNetwork(tt.args.ctx, tt.args.networkID); (err != nil) != tt.wantErr {
				t.Errorf("Client.RemoveNetwork() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_PruneNetworks(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		c       *Client
		args    args
		want    network.PruneReport
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.c.PruneNetworks(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.PruneNetworks() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Client.PruneNetworks() = %v, want %v", got, tt.want)
			}
		})
	}
}
