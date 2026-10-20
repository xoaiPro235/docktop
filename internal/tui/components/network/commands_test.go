package network

import (
	"context"
	"errors"
	"reflect"
	"testing"

	mobyNetwork "github.com/moby/moby/api/types/network"
	"github.com/xoaiPro235/docktop/internal/docker/mocks"
	"github.com/xoaiPro235/docktop/internal/tui/modal"
)

func TestModel_removeNetworkCmd(t *testing.T) {
	tests := []struct {
		name       string
		networkID  string
		mockErr    error
		wantAction modal.Action
		wantErr    bool
	}{
		{
			name:       "successful network removal",
			networkID:  "net-custom",
			mockErr:    nil,
			wantAction: modal.ActionDelete,
			wantErr:    false,
		},
		{
			name:       "network in use error",
			networkID:  "net-busy",
			mockErr:    errors.New("network has active endpoints"),
			wantAction: modal.ActionDelete,
			wantErr:    true,
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
			actionMsg, ok := msg.(NetworkActionMsg)
			if !ok {
				t.Fatalf("Expected NetworkActionMsg, got %T", msg)
			}
			if !called {
				t.Error("Mock NetworkService.RemoveNetwork was not called")
			}
			if actionMsg.Action != tt.wantAction {
				t.Errorf("Action = %v, want %v", actionMsg.Action, tt.wantAction)
			}
			if (actionMsg.Err != nil) != tt.wantErr {
				t.Errorf("Err = %v, wantErr = %v", actionMsg.Err, tt.wantErr)
			}
		})
	}
}

func TestModel_pruneNetworksCmd(t *testing.T) {
	tests := []struct {
		name       string
		report     mobyNetwork.PruneReport
		mockErr    error
		wantAction modal.Action
		wantErr    bool
	}{
		{
			name: "successful network prune",
			report: mobyNetwork.PruneReport{
				NetworksDeleted: []string{"net1", "net2"},
			},
			mockErr:    nil,
			wantAction: modal.ActionPrune,
			wantErr:    false,
		},
		{
			name:       "error on prune",
			report:     mobyNetwork.PruneReport{},
			mockErr:    errors.New("daemon error"),
			wantAction: modal.ActionPrune,
			wantErr:    true,
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
			actionMsg, ok := msg.(NetworkActionMsg)
			if !ok {
				t.Fatalf("Expected NetworkActionMsg, got %T", msg)
			}
			if !called {
				t.Error("Mock NetworkService.PruneNetworks was not called")
			}
			if !reflect.DeepEqual(actionMsg.Action, tt.wantAction) {
				t.Errorf("Action = %v, want %v", actionMsg.Action, tt.wantAction)
			}
			if (actionMsg.Err != nil) != tt.wantErr {
				t.Errorf("Err = %v, wantErr = %v", actionMsg.Err, tt.wantErr)
			}
		})
	}
}
