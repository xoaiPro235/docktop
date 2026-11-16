package network

import (
	"context"
	"errors"
	"testing"

	mobyNetwork "github.com/moby/moby/api/types/network"
	"github.com/xoaiPro235/docktop/internal/docker/mocks"
)

func TestModel_removeNetworkCmd(t *testing.T) {
	tests := []struct {
		name      string
		networkID string
		mockErr   error
		wantErr   bool
	}{
		{
			name:      "successful network removal",
			networkID: "net-custom",
			mockErr:   nil,
			wantErr:   false,
		},
		{
			name:      "network in use error",
			networkID: "net-busy",
			mockErr:   errors.New("network has active endpoints"),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			mockCli := &mocks.MockNetworkService{
				RemoveNetworkFunc: func(ctx context.Context, id string) error {
					called = true
					if id != tt.networkID {
						t.Errorf("RemoveNetwork id = %v, want %v", id, tt.networkID)
					}
					return tt.mockErr
				},
			}

			m := New(mockCli)
			cmd := m.removeNetworkCmd(tt.networkID)
			if cmd == nil {
				t.Fatal("removeNetworkCmd returned nil")
			}

			msg := cmd()
			resultMsg, ok := msg.(NetworkResultMsg)
			if !ok {
				t.Fatalf("Expected NetworkResultMsg, got %T", msg)
			}
			if !called {
				t.Error("Mock NetworkService.RemoveNetwork was not called")
			}
			if resultMsg.TargetID != tt.networkID {
				t.Errorf("TargetID = %v, want %v", resultMsg.TargetID, tt.networkID)
			}
			if (resultMsg.Err != nil) != tt.wantErr {
				t.Errorf("Err = %v, wantErr = %v", resultMsg.Err, tt.wantErr)
			}
		})
	}
}

func TestModel_pruneNetworksCmd(t *testing.T) {
	tests := []struct {
		name    string
		report  mobyNetwork.PruneReport
		mockErr error
		wantErr bool
	}{
		{
			name: "successful network prune",
			report: mobyNetwork.PruneReport{
				NetworksDeleted: []string{"net1", "net2"},
			},
			mockErr: nil,
			wantErr: false,
		},
		{
			name:    "error on prune",
			report:  mobyNetwork.PruneReport{},
			mockErr: errors.New("daemon error"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			mockCli := &mocks.MockNetworkService{
				PruneNetworksFunc: func(ctx context.Context) (mobyNetwork.PruneReport, error) {
					called = true
					return tt.report, tt.mockErr
				},
			}

			m := New(mockCli)
			cmd := m.pruneNetworksCmd()
			if cmd == nil {
				t.Fatal("pruneNetworksCmd returned nil")
			}

			msg := cmd()
			resultMsg, ok := msg.(NetworkResultMsg)
			if !ok {
				t.Fatalf("Expected NetworkResultMsg, got %T", msg)
			}
			if !called {
				t.Error("Mock NetworkService.PruneNetworks was not called")
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
