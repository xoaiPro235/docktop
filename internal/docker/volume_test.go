package docker

import (
	"context"
	"reflect"
	"testing"

	"github.com/moby/moby/api/types/volume"
)

func TestClient_ListVolumes(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		c       *Client
		args    args
		want    []VolumeInfo
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.c.ListVolumes(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.ListVolumes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Client.ListVolumes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_InspectVolume(t *testing.T) {
	type args struct {
		ctx  context.Context
		name string
	}
	tests := []struct {
		name    string
		c       *Client
		args    args
		want    *volume.Volume
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.c.InspectVolume(tt.args.ctx, tt.args.name)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.InspectVolume() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Client.InspectVolume() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestClient_RemoveVolume(t *testing.T) {
	type args struct {
		ctx   context.Context
		name  string
		force bool
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
			if err := tt.c.RemoveVolume(tt.args.ctx, tt.args.name, tt.args.force); (err != nil) != tt.wantErr {
				t.Errorf("Client.RemoveVolume() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClient_PruneVolumes(t *testing.T) {
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		c       *Client
		args    args
		want    volume.PruneReport
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.c.PruneVolumes(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.PruneVolumes() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Client.PruneVolumes() = %v, want %v", got, tt.want)
			}
		})
	}
}
