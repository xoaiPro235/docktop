package container

import (
	"context"
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
	mockCli := &mocks.MockContainerService{}
	m := New(mockCli)

	if m.dockerClient == nil {
		t.Error("Expected dockerClient to be assigned")
	}
	if m.focusZone != FocusLeftTree {
		t.Errorf("focusZone = %v, want FocusLeftTree", m.focusZone)
	}
	if m.activePane != SubPaneStats {
		t.Errorf("activePane = %v, want SubPaneStats", m.activePane)
	}
	if m.IsModalActive() {
		t.Error("New model should have inactive modal")
	}
}

func TestModel_Init(t *testing.T) {
	mockCli := &mocks.MockContainerService{}
	m := New(mockCli)
	if cmd := m.Init(); cmd != nil {
		t.Errorf("Init() = %v, want nil", cmd)
	}
}

func TestModel_SelectedNodeAndContainer(t *testing.T) {
	mockCli := &mocks.MockContainerService{}
	m := New(mockCli)

	if m.SelectedNode() != nil {
		t.Error("SelectedNode() should be nil on empty list")
	}
	if m.SelectedContainer() != nil {
		t.Error("SelectedContainer() should be nil on empty list")
	}

	ctr := &docker.ContainerInfo{ID: "c1", Name: "web-server"}
	node := &TreeNode{
		Type:      NodeContainer,
		Name:      "web-server",
		Container: ctr,
	}
	m.flatItems = []FlatItem{{Node: node}}
	m.cursorIndex = 0

	if m.SelectedNode() != node {
		t.Errorf("SelectedNode() = %v, want %v", m.SelectedNode(), node)
	}
	if m.SelectedContainer() != ctr {
		t.Errorf("SelectedContainer() = %v, want %v", m.SelectedContainer(), ctr)
	}
}

func TestModel_SetContainers(t *testing.T) {
	mockCli := &mocks.MockContainerService{
		StreamContainerStatsFunc: func(ctx context.Context, id string) <-chan docker.ContainerMetrics {
			return make(chan docker.ContainerMetrics)
		},
	}
	m := New(mockCli)
	m.SetSize(120, 40)

	list := []docker.ContainerInfo{
		{ID: "c1", Name: "app1", IsCompose: false, State: docker.StateRunning},
		{ID: "c2", Name: "app2", IsCompose: false, State: docker.StateRunning},
	}

	m.SetContainers(list)
	if len(m.containers) != 2 {
		t.Fatalf("len(containers) = %d, want 2", len(m.containers))
	}
	if len(m.flatItems) == 0 {
		t.Fatal("flatItems should be populated after SetContainers")
	}
}

func TestModel_PauseAndResume(t *testing.T) {
	mockCli := &mocks.MockContainerService{}
	m := New(mockCli)

	cancelCalled := false
	m.logCancel = func() { cancelCalled = true }
	m.Pause()
	if !cancelCalled {
		t.Error("Pause() should call logCancel")
	}

	// Resume should not panic
	m.Resume()
}

func TestModel_IsModalActive(t *testing.T) {
	mockCli := &mocks.MockContainerService{}
	m := New(mockCli)
	if m.IsModalActive() {
		t.Error("IsModalActive() should be false initially")
	}

	m.confirmModal.ShowContainerAction(modal.ActionRestart, "c1", "web")
	if !m.IsModalActive() {
		t.Error("IsModalActive() should be true when modal is shown")
	}
}

func TestModel_ActionsRunning(t *testing.T) {
	mockCli := &mocks.MockContainerService{}
	m := New(mockCli)

	if m.IsActionRunning() {
		t.Error("IsActionRunning() should be false initially")
	}
	if m.isContainerActionRunning("c1") {
		t.Error("isContainerActionRunning(c1) should be false initially")
	}

	m.runningActions["c1"] = RunningAction{Action: modal.ActionStop, TargetName: "web"}
	if !m.IsActionRunning() {
		t.Error("IsActionRunning() should be true when runningActions has items")
	}
	if !m.isContainerActionRunning("c1") {
		t.Error("isContainerActionRunning(c1) should be true")
	}
}

func TestModel_SetSize(t *testing.T) {
	mockCli := &mocks.MockContainerService{}
	m := New(mockCli)
	m.SetSize(140, 45)

	if m.width != 140 || m.height != 45 {
		t.Errorf("width, height = (%d, %d), want (140, 45)", m.width, m.height)
	}
}

func TestModel_Streams(t *testing.T) {
	metricsCh := make(chan docker.ContainerMetrics, 1)
	logCh := make(chan string, 1)

	mockCli := &mocks.MockContainerService{
		StreamContainerStatsFunc: func(ctx context.Context, id string) <-chan docker.ContainerMetrics {
			return metricsCh
		},
		StreamContainerLogsFunc: func(ctx context.Context, containerID string, tail string) <-chan string {
			return logCh
		},
	}

	m := New(mockCli)
	m.SetSize(100, 30)

	// Start stats stream
	m.startStatsStream("c1")
	if m.statsChan == nil {
		t.Error("statsChan should not be nil after startStatsStream")
	}
	m.stopStatsStream()
	if m.statsCancel != nil {
		t.Error("statsCancel should be nil after stopStatsStream")
	}

	// Start log stream
	m.startLogStream("c1")
	if m.logChan == nil {
		t.Error("logChan should not be nil after startLogStream")
	}
	m.stopLogStream()
	if m.logCancel != nil {
		t.Error("logCancel should be nil after stopLogStream")
	}
}
