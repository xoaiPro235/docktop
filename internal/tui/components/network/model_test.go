package network

import (
	"testing"

	"github.com/xoaiPro235/docktop/internal/docker"
	"github.com/xoaiPro235/docktop/internal/docker/mocks"
	"github.com/xoaiPro235/docktop/internal/tui/modal"
	"github.com/xoaiPro235/docktop/internal/tui/theme"
)

func init() {
	theme.SetThemeByName("tokyo-night")
}

func TestNew(t *testing.T) {
	mockCli := &mocks.MockNetworkService{}
	m := New(mockCli)

	if m.dockerClient == nil {
		t.Error("Expected dockerClient to be assigned")
	}
	if m.focusZone != FocusNetworksList {
		t.Errorf("focusZone = %v, want FocusNetworksList", m.focusZone)
	}
	if m.IsModalActive() {
		t.Error("New model should have inactive modal")
	}
	if m.IsActionRunning() {
		t.Error("New model should not have active action")
	}
}

func TestModel_Init(t *testing.T) {
	mockCli := &mocks.MockNetworkService{}
	m := New(mockCli)
	if cmd := m.Init(); cmd != nil {
		t.Errorf("Init() = %v, want nil", cmd)
	}
}

func TestModel_SetSize(t *testing.T) {
	mockCli := &mocks.MockNetworkService{}
	m := New(mockCli)
	m.SetSize(120, 40)

	if m.width != 120 || m.height != 40 {
		t.Errorf("width, height = (%d, %d), want (120, 40)", m.width, m.height)
	}
	if !m.viewportReady {
		t.Error("viewportReady should be true after SetSize")
	}
}

func TestModel_SetNetworks(t *testing.T) {
	mockCli := &mocks.MockNetworkService{}
	m := New(mockCli)
	m.SetSize(100, 30)

	nets := []docker.NetworkInfo{
		{ID: "net-1", Name: "bridge"},
		{ID: "net-2", Name: "custom_app"},
		{ID: "net-3", Name: "host"},
	}

	m.SetNetworks(nets)
	if len(m.networks) != 3 {
		t.Fatalf("len(networks) = %d, want 3", len(m.networks))
	}
	if m.SelectedNetwork() == nil || m.SelectedNetwork().ID != "net-1" {
		t.Errorf("SelectedNetwork = %v, want net-1", m.SelectedNetwork())
	}

	// Move cursor to net-2 and update list
	m.cursorIndex = 1
	newNets := []docker.NetworkInfo{
		{ID: "net-3", Name: "host"},
		{ID: "net-2", Name: "custom_app"},
	}
	m.SetNetworks(newNets)
	if m.SelectedNetwork() == nil || m.SelectedNetwork().ID != "net-2" {
		t.Errorf("SelectedNetwork after re-order = %v, want net-2", m.SelectedNetwork())
	}
}

func TestModel_IsModalActive(t *testing.T) {
	mockCli := &mocks.MockNetworkService{}
	m := New(mockCli)
	if m.IsModalActive() {
		t.Error("IsModalActive() should be false initially")
	}

	m.confirmModal.ShowNetworkAction(modal.ActionDelete, "net-1", "custom_net")
	if !m.IsModalActive() {
		t.Error("IsModalActive() should be true when modal is shown")
	}
}

func TestModel_IsActionRunning(t *testing.T) {
	mockCli := &mocks.MockNetworkService{}
	m := New(mockCli)
	if m.IsActionRunning() {
		t.Error("IsActionRunning() should be false initially")
	}

	m.action.Start(modal.ActionDelete, "net-1", "custom_net", "Deleting network")
	if !m.IsActionRunning() {
		t.Error("IsActionRunning() should be true after action.Start")
	}
}

func TestModel_SelectedNetwork(t *testing.T) {
	mockCli := &mocks.MockNetworkService{}
	m := New(mockCli)

	if got := m.SelectedNetwork(); got != nil {
		t.Errorf("SelectedNetwork() on empty list = %v, want nil", got)
	}

	m.networks = []docker.NetworkInfo{
		{ID: "net-a", Name: "bridge"},
		{ID: "net-b", Name: "overlay"},
	}
	m.cursorIndex = 1

	if got := m.SelectedNetwork(); got == nil || got.ID != "net-b" {
		t.Errorf("SelectedNetwork() = %v, want net-b", got)
	}
}

func TestModel_updateViewportContent(t *testing.T) {
	mockCli := &mocks.MockNetworkService{}
	m := New(mockCli)

	// Before viewport is ready
	m.updateViewportContent()

	// After viewport is ready
	m.SetSize(100, 30)
	m.updateViewportContent()

	// With populated networks
	m.networks = []docker.NetworkInfo{
		{
			ID:             "net-123456",
			Name:           "docktop_default",
			Driver:         "bridge",
			Scope:          "local",
			Subnet:         "172.20.0.0/16",
			Gateway:        "172.20.0.1",
			ConnectedCount: 1,
			Containers: []docker.NetworkEndpointInfo{
				{ContainerID: "cid1", Name: "web-server", IP: "172.20.0.2", Ports: "80:80"},
			},
		},
	}
	m.updateViewportContent()
}
