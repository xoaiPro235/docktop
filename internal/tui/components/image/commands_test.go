package image

import (
	"context"
	"errors"
	"testing"

	mobyimage "github.com/moby/moby/api/types/image"
	"github.com/xoaiPro235/docktop/internal/docker"
	"github.com/xoaiPro235/docktop/internal/docker/mocks"
)

func TestModel_fetchImageDetailsCmd(t *testing.T) {
	tests := []struct {
		name        string
		imageID     string
		mockDetails *mobyimage.InspectResponse
		mockLayers  []docker.ImageLayer
		mockErr     error
	}{
		{
			name:        "successful fetch details and history",
			imageID:     "sha256:12345",
			mockDetails: &mobyimage.InspectResponse{ID: "sha256:12345"},
			mockLayers: []docker.ImageLayer{
				{Index: 0, Command: "CMD [/app]"},
			},
			mockErr: nil,
		},
		{
			name:        "error fetching history",
			imageID:     "sha256:fail",
			mockDetails: nil,
			mockLayers:  nil,
			mockErr:     errors.New("failed to get history"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCli := &mocks.MockImageService{
				InspectImageFunc: func(ctx context.Context, id string) (*mobyimage.InspectResponse, error) {
					return tt.mockDetails, nil
				},
				GetImageHistoryFunc: func(ctx context.Context, id string) ([]docker.ImageLayer, error) {
					return tt.mockLayers, tt.mockErr
				},
			}

			m := New(mockCli)
			cmd := m.fetchImageDetailsCmd(tt.imageID)
			if cmd == nil {
				t.Fatal("fetchImageDetailsCmd returned nil")
			}

			msg := cmd()
			detailsMsg, ok := msg.(ImageDetailsMsg)
			if !ok {
				t.Fatalf("Expected ImageDetailsMsg, got %T", msg)
			}
			if detailsMsg.ImageID != tt.imageID {
				t.Errorf("ImageID = %v, want %v", detailsMsg.ImageID, tt.imageID)
			}
			if (detailsMsg.Err != nil) != (tt.mockErr != nil) {
				t.Errorf("Err = %v, wantErr = %v", detailsMsg.Err, tt.mockErr != nil)
			}
		})
	}
}

func TestModel_removeImageCmd(t *testing.T) {
	tests := []struct {
		name     string
		targetID string
		mockErr  error
		wantErr  bool
	}{
		{
			name:     "successful removal",
			targetID: "img-1",
			mockErr:  nil,
			wantErr:  false,
		},
		{
			name:     "removal conflict error",
			targetID: "img-in-use",
			mockErr:  errors.New("image is being used by running container"),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			mockCli := &mocks.MockImageService{
				RemoveImageFunc: func(ctx context.Context, id string, force bool) error {
					called = true
					if id != tt.targetID {
						t.Errorf("RemoveImage targetID = %v, want %v", id, tt.targetID)
					}
					return tt.mockErr
				},
			}

			m := New(mockCli)
			cmd := m.removeImageCmd(tt.targetID)
			if cmd == nil {
				t.Fatal("removeImageCmd returned nil")
			}

			msg := cmd()
			resultMsg, ok := msg.(ImageResultMsg)
			if !ok {
				t.Fatalf("Expected ImageResultMsg, got %T", msg)
			}
			if !called {
				t.Error("Mock ImageService.RemoveImage was not called")
			}
			if resultMsg.TargetID != tt.targetID {
				t.Errorf("TargetID = %v, want %v", resultMsg.TargetID, tt.targetID)
			}
			if (resultMsg.Err != nil) != tt.wantErr {
				t.Errorf("Err = %v, wantErr = %v", resultMsg.Err, tt.wantErr)
			}
		})
	}
}

func TestModel_pruneImagesCmd(t *testing.T) {
	tests := []struct {
		name    string
		report  mobyimage.PruneReport
		mockErr error
		wantErr bool
	}{
		{
			name: "successful prune",
			report: mobyimage.PruneReport{
				ImagesDeleted: []mobyimage.DeleteResponse{{Untagged: "dangling"}},
			},
			mockErr: nil,
			wantErr: false,
		},
		{
			name:    "prune error",
			report:  mobyimage.PruneReport{},
			mockErr: errors.New("daemon connection lost"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			mockCli := &mocks.MockImageService{
				PruneImagesFunc: func(ctx context.Context) (mobyimage.PruneReport, error) {
					called = true
					return tt.report, tt.mockErr
				},
			}

			m := New(mockCli)
			cmd := m.pruneImagesCmd()
			if cmd == nil {
				t.Fatal("pruneImagesCmd returned nil")
			}

			msg := cmd()
			resultMsg, ok := msg.(ImageResultMsg)
			if !ok {
				t.Fatalf("Expected ImageResultMsg, got %T", msg)
			}
			if !called {
				t.Error("Mock ImageService.PruneImages was not called")
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
