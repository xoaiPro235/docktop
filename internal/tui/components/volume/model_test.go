package volume

import (
	"testing"

	"github.com/moby/moby/api/types/volume"
	"github.com/xoaiPro235/docktop/internal/docker"
	"github.com/xoaiPro235/docktop/internal/docker/mocks"
	"github.com/xoaiPro235/docktop/internal/tui/modal"
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
	if m.IsModalActive() {
		t.Error("New model should have inactive modal")
	}
	if m.IsActionRunning() {
		t.Error("New model should not have active action")
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

func TestModel_IsModalActive(t *testing.T) {
	mockCli := &mocks.MockVolumeService{}
	m := New(mockCli)
	if m.IsModalActive() {
		t.Error("IsModalActive() should be false initially")
	}

	m.confirmModal.ShowVolumeAction(modal.ActionRemove, "vol-1")
	if !m.IsModalActive() {
		t.Error("IsModalActive() should be true when modal is shown")
	}
}

func TestModel_IsActionRunning(t *testing.T) {
	mockCli := &mocks.MockVolumeService{}
	m := New(mockCli)
	if m.IsActionRunning() {
		t.Error("IsActionRunning() should be false initially")
	}

	m.action.Start(modal.ActionRemove, "vol-1", "vol-1", "Removing volume")
	if !m.IsActionRunning() {
		t.Error("IsActionRunning() should be true after action.Start")
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
	m.updateViewportContent()
}
