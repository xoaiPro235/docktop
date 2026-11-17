package network

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/xoaiPro235/docktop/internal/docker"
	"github.com/xoaiPro235/docktop/internal/docker/mocks"
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
	if m.IsLoading() {
		t.Error("New model should not be loading")
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

func TestModel_FilterNetworks(t *testing.T) {
	m := New(&mocks.MockNetworkService{})
	m.SetNetworks([]docker.NetworkInfo{
		{ID: "net-1", Name: "bridge", Driver: "bridge", Subnet: "172.17.0.0/16"},
		{ID: "net-2", Name: "app", Driver: "bridge", Subnet: "172.20.0.0/16"},
	})

	sendKey := func(msg tea.Msg) {
		m, _ = m.Update(msg)
	}
	sendKey(tea.KeyPressMsg{Code: '/', Text: "/"})
	for _, r := range "172.20" {
		sendKey(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	sendKey(tea.KeyPressMsg{Code: tea.KeyEnter})

	if m.IsSearching() {
		t.Error("search input should be inactive after submitting the filter")
	}
	if got := m.SelectedNetwork(); got == nil || got.ID != "net-2" {
		t.Errorf("SelectedNetwork() after filtering = %v, want net-2", got)
	}
	if len(m.filteredIndices) != 1 {
		t.Errorf("len(filteredIndices) after filtering = %d, want 1", len(m.filteredIndices))
	}

	sendKey(tea.KeyPressMsg{Code: tea.KeyEsc})
	if len(m.filteredIndices) != 2 {
		t.Errorf("len(filteredIndices) after clearing filter = %d, want 2", len(m.filteredIndices))
	}
}

func TestModel_IsLoading(t *testing.T) {
	mockCli := &mocks.MockNetworkService{}
	m := New(mockCli)
	if m.IsLoading() {
		t.Error("IsLoading() should be false initially")
	}

	m.runningIDs["net-1"] = true
	if !m.IsLoading() {
		t.Error("IsLoading() should be true when runningIDs is set")
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
	m.applyFilter()
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

func TestModel_ZoomToggle(t *testing.T) {
	mockCli := &mocks.MockNetworkService{}
	m := New(mockCli)
	m.SetSize(100, 30)

	if m.IsZoomed() {
		t.Fatal("expected initially unzoomed")
	}
	origLeft := m.leftWidth()
	origRight := m.rightWidth()

	// 1. Press 'z' to zoom left pane
	m, _ = m.Update(tea.KeyPressMsg{Code: 'z', Text: "z"})
	if !m.IsZoomed() {
		t.Fatal("expected zoomed after 'z'")
	}
	if !m.isLeftZoomed() {
		t.Fatal("expected left pane to be zoomed")
	}
	if m.leftWidth() != 100 {
		t.Errorf("expected leftWidth=100, got %d", m.leftWidth())
	}
	if m.rightWidth() != 0 {
		t.Errorf("expected rightWidth=0, got %d", m.rightWidth())
	}

	// 2. Tab to switch focus to right pane while zoomed
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if !m.isRightZoomed() {
		t.Fatal("expected right pane to be zoomed after tab")
	}
	if m.leftWidth() != 0 {
		t.Errorf("expected leftWidth=0, got %d", m.leftWidth())
	}
	if m.rightWidth() != 100 {
		t.Errorf("expected rightWidth=100, got %d", m.rightWidth())
	}

	// 3. Press 'z' again to unzoom
	m, _ = m.Update(tea.KeyPressMsg{Code: 'z', Text: "z"})
	if m.IsZoomed() {
		t.Fatal("expected unzoomed after second 'z'")
	}
	if m.leftWidth() != origLeft || m.rightWidth() != origRight {
		t.Errorf("expected restored widths (%d, %d), got (%d, %d)", origLeft, origRight, m.leftWidth(), m.rightWidth())
	}

	// 4. When search bar is active, 'z' should NOT toggle zoom
	_ = m.searchBar.Activate()
	m, _ = m.Update(tea.KeyPressMsg{Code: 'z', Text: "z"})
	if m.IsZoomed() {
		t.Fatal("expected zoom not to toggle when search bar is active")
	}
	if m.searchBar.Value() != "z" {
		t.Errorf("expected searchBar value 'z', got %q", m.searchBar.Value())
	}
}

func TestFormatNetworkViewportContent_IPv6AndZoom(t *testing.T) {
	net := &docker.NetworkInfo{
		ID:     "net1",
		Name:   "test-net",
		Driver: "bridge",
		Containers: []docker.NetworkEndpointInfo{
			{
				Name:  "my-super-long-container-name-example-worker",
				IP:    "2001:db8:1234:5678::2/64",
				Ports: "8080:80",
			},
		},
	}

	// Unzoomed: truncates with ellipsis so columns stay aligned and PORTS is visible
	unzoomedContent := FormatNetworkViewportContent(net, 60, false)
	if !strings.Contains(unzoomedContent, "…") {
		t.Errorf("expected ellipsis in unzoomed content, got: %s", unzoomedContent)
	}

	// Zoomed: shows complete IPv6 address
	zoomedContent := FormatNetworkViewportContent(net, 120, true)
	if !strings.Contains(zoomedContent, "2001:db8:1234:5678::2/64") {
		t.Errorf("expected full IPv6 address in zoomed content, got: %s", zoomedContent)
	}
}
