package volume

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/moby/moby/api/types/volume"
	"github.com/xoaiPro235/docktop/internal/docker"
	"github.com/xoaiPro235/docktop/internal/docker/mocks"
	"github.com/xoaiPro235/docktop/internal/tui/theme"
)

func init() {
	theme.SetThemeByName("tokyo-night")
}

func TestNew(t *testing.T) {
	mockCli := &mocks.MockVolumeService{}
	m := New(mockCli)

	if m.dockerClient == nil {
		t.Error("Expected dockerClient to be assigned")
	}
	if m.focusZone != FocusVolumesList {
		t.Errorf("focusZone = %v, want FocusVolumesList", m.focusZone)
	}
	if m.IsLoading() {
		t.Error("New model should not be loading")
	}
}

func TestModel_Init(t *testing.T) {
	mockCli := &mocks.MockVolumeService{}
	m := New(mockCli)
	if cmd := m.Init(); cmd != nil {
		t.Errorf("Init() = %v, want nil", cmd)
	}
}

func TestModel_SetSize(t *testing.T) {
	mockCli := &mocks.MockVolumeService{}
	m := New(mockCli)
	m.SetSize(120, 40)

	if m.width != 120 || m.height != 40 {
		t.Errorf("width, height = (%d, %d), want (120, 40)", m.width, m.height)
	}
	if !m.viewportReady {
		t.Error("viewportReady should be true after SetSize")
	}
}

func TestModel_SetVolumes(t *testing.T) {
	mockCli := &mocks.MockVolumeService{}
	m := New(mockCli)
	m.SetSize(100, 30)

	vols := []docker.VolumeInfo{
		{Volume: volume.Volume{Name: "data-1"}},
		{Volume: volume.Volume{Name: "data-2"}},
		{Volume: volume.Volume{Name: "data-3"}},
	}

	m.SetVolumes(vols)
	if len(m.volumes) != 3 {
		t.Fatalf("len(volumes) = %d, want 3", len(m.volumes))
	}
	if m.SelectedVolume() == nil || m.SelectedVolume().Name != "data-1" {
		t.Errorf("SelectedVolume = %v, want data-1", m.SelectedVolume())
	}

	// Move cursor to data-2 and update list
	m.cursorIndex = 1
	newVols := []docker.VolumeInfo{
		{Volume: volume.Volume{Name: "data-3"}},
		{Volume: volume.Volume{Name: "data-2"}},
	}
	m.SetVolumes(newVols)
	if m.SelectedVolume() == nil || m.SelectedVolume().Name != "data-2" {
		t.Errorf("SelectedVolume after re-order = %v, want data-2", m.SelectedVolume())
	}
}

func TestModel_FilterVolumes(t *testing.T) {
	m := New(&mocks.MockVolumeService{})
	m.SetVolumes([]docker.VolumeInfo{
		{Volume: volume.Volume{Name: "pgdata", Driver: "local"}},
		{Volume: volume.Volume{Name: "cache", Driver: "local"}},
	})

	sendKey := func(msg tea.Msg) {
		m, _ = m.Update(msg)
	}
	sendKey(tea.KeyPressMsg{Code: '/', Text: "/"})
	for _, r := range "PGDATA" {
		sendKey(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	sendKey(tea.KeyPressMsg{Code: tea.KeyEnter})

	if m.IsSearching() {
		t.Error("search input should be inactive after submitting the filter")
	}
	if got := m.SelectedVolume(); got == nil || got.Name != "pgdata" {
		t.Errorf("SelectedVolume() after filtering = %v, want pgdata", got)
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
	mockCli := &mocks.MockVolumeService{}
	m := New(mockCli)
	if m.IsLoading() {
		t.Error("IsLoading() should be false initially")
	}

	m.runningIDs["vol-1"] = true
	if !m.IsLoading() {
		t.Error("IsLoading() should be true when runningIDs is set")
	}
}

func TestModel_SelectedVolume(t *testing.T) {
	mockCli := &mocks.MockVolumeService{}
	m := New(mockCli)

	if got := m.SelectedVolume(); got != nil {
		t.Errorf("SelectedVolume() on empty list = %v, want nil", got)
	}

	m.volumes = []docker.VolumeInfo{
		{Volume: volume.Volume{Name: "vol-alpha"}},
		{Volume: volume.Volume{Name: "vol-beta"}},
	}
	m.applyFilter()
	m.cursorIndex = 1

	if got := m.SelectedVolume(); got == nil || got.Name != "vol-beta" {
		t.Errorf("SelectedVolume() = %v, want vol-beta", got)
	}
}

func TestModel_updateViewportContent(t *testing.T) {
	mockCli := &mocks.MockVolumeService{}
	m := New(mockCli)

	// Before viewport is ready, should not panic
	m.updateViewportContent()

	// After viewport is ready
	m.SetSize(100, 30)
	m.updateViewportContent()

	// With populated volumes
	m.volumes = []docker.VolumeInfo{
		{
			Volume: volume.Volume{
				Name:       "pgdata",
				Driver:     "local",
				Mountpoint: "/var/lib/docker/volumes/pgdata/_data",
				Scope:      "local",
			},
			InUse:  true,
			RefCnt: 1,
			Containers: []docker.VolumeContainerMount{
				{ContainerName: "postgres-db", Destination: "/var/lib/postgresql/data", RW: true},
			},
		},
	}
	m.applyFilter()
	m.updateViewportContent()
}

func TestModel_VolumeNamedAllNotPrune(t *testing.T) {
	m := New(&mocks.MockVolumeService{})
	m.runningIDs["all"] = true
	if m.IsPruning() {
		t.Error("a volume named 'all' in runningIDs should not trigger IsPruning()")
	}
	if m.isVolumeActionRunning("other-volume") {
		t.Error("a volume named 'all' in runningIDs should not block other-volume")
	}

	m.runningIDs[ActionTargetAll] = true
	if !m.IsPruning() {
		t.Error("ActionTargetAll in runningIDs should trigger IsPruning()")
	}
	if !m.isVolumeActionRunning("other-volume") {
		t.Error("ActionTargetAll in runningIDs should block other-volume")
	}
}

func TestModel_RejectDuplicateRunCmdMsg(t *testing.T) {
	m := New(&mocks.MockVolumeService{})
	dummyCmd := func() tea.Msg { return nil }

	var cmd tea.Cmd
	m, cmd = m.Update(RunCmdMsg{TargetID: "vol-1", Cmd: dummyCmd})
	if cmd == nil {
		t.Fatal("first RunCmdMsg should return a command")
	}

	_, cmd = m.Update(RunCmdMsg{TargetID: "vol-1", Cmd: dummyCmd})
	if cmd != nil {
		t.Errorf("duplicate RunCmdMsg for running target should return nil cmd, got %v", cmd)
	}
}

func TestModel_ZoomToggle(t *testing.T) {
	mockCli := &mocks.MockVolumeService{}
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
