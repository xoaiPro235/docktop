package volume

import (
	"context"
	"errors"
	"testing"

	"github.com/moby/moby/api/types/volume"
	"github.com/xoaiPro235/docktop/internal/docker/mocks"
)

func TestModel_removeVolumeCmd(t *testing.T) {
	tests := []struct {
		name       string
		volumeName string
		force      bool
		mockErr    error
		wantErr    bool
	}{
		{
			name:       "successful remove",
			volumeName: "vol-1",
			force:      false,
			mockErr:    nil,
			wantErr:    false,
		},
		{
			name:       "remove error",
			volumeName: "vol-in-use",
			force:      true,
			mockErr:    errors.New("volume is in use"),
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			mockCli := &mocks.MockVolumeService{
				RemoveVolumeFunc: func(ctx context.Context, name string, force bool) error {
					called = true
					if name != tt.volumeName {
						t.Errorf("RemoveVolume name = %v, want %v", name, tt.volumeName)
					}
					if force != tt.force {
						t.Errorf("RemoveVolume force = %v, want %v", force, tt.force)
					}
					return tt.mockErr
				},
			}

			m := New(mockCli)
			cmd := m.removeVolumeCmd(tt.volumeName, tt.force)
			if cmd == nil {
				t.Fatal("removeVolumeCmd returned nil tea.Cmd")
			}

			msg := cmd()
			resultMsg, ok := msg.(VolumeResultMsg)
			if !ok {
				t.Fatalf("Expected VolumeResultMsg, got %T", msg)
			}
			if !called {
				t.Error("Mock VolumeService.RemoveVolume was not called")
			}
			if resultMsg.TargetID != tt.volumeName {
				t.Errorf("TargetID = %v, want %v", resultMsg.TargetID, tt.volumeName)
			}
			if (resultMsg.Err != nil) != tt.wantErr {
				t.Errorf("Err = %v, wantErr = %v", resultMsg.Err, tt.wantErr)
			}
		})
	}
}

func TestModel_pruneVolumesCmd(t *testing.T) {
	tests := []struct {
		name    string
		report  volume.PruneReport
		mockErr error
		wantErr bool
	}{
		{
			name: "successful prune",
			report: volume.PruneReport{
				VolumesDeleted: []string{"v1", "v2"},
				SpaceReclaimed: 1024,
			},
			mockErr: nil,
			wantErr: false,
		},
		{
			name:    "prune error",
			report:  volume.PruneReport{},
			mockErr: errors.New("daemon failure"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			mockCli := &mocks.MockVolumeService{
				PruneVolumesFunc: func(ctx context.Context) (volume.PruneReport, error) {
					called = true
					return tt.report, tt.mockErr
				},
			}

			m := New(mockCli)
			cmd := m.pruneVolumesCmd()
			if cmd == nil {
				t.Fatal("pruneVolumesCmd returned nil tea.Cmd")
			}

			msg := cmd()
			resultMsg, ok := msg.(VolumeResultMsg)
			if !ok {
				t.Fatalf("Expected VolumeResultMsg, got %T", msg)
			}
			if !called {
				t.Error("Mock VolumeService.PruneVolumes was not called")
			}
			if resultMsg.TargetID != ActionTargetAll {
				t.Errorf("TargetID = %v, want %v", resultMsg.TargetID, ActionTargetAll)
			}
			if (resultMsg.Err != nil) != tt.wantErr {
				t.Errorf("Err = %v, wantErr = %v", resultMsg.Err, tt.wantErr)
			}
		})
	}
}
