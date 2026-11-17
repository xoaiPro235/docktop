package container

import (
	"context"
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

func TestModel_FilterContainers(t *testing.T) {
	m := New(&mocks.MockContainerService{})
	m.updateContainers([]docker.ContainerInfo{
		{ID: "c1", Name: "cache", Image: "redis:7", State: docker.StateExited},
		{ID: "c2", Name: "web", Image: "nginx:latest", State: docker.StateExited},
	})

	sendKey := func(msg tea.Msg) {
		m, _ = m.Update(msg)
	}
	sendKey(tea.KeyPressMsg{Code: '/', Text: "/"})
	for _, r := range "REDIS:7" {
		sendKey(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	sendKey(tea.KeyPressMsg{Code: tea.KeyEnter})

	if m.IsSearching() {
		t.Error("search input should be inactive after submitting the filter")
	}
	if got := m.SelectedContainer(); got == nil || got.ID != "c1" {
		t.Errorf("SelectedContainer() after filtering = %v, want c1", got)
	}
	countContainers := func() int {
		count := 0
		for _, item := range m.flatItems {
			if item.Node.Container != nil {
				count++
			}
		}
		return count
	}
	if got := countContainers(); got != 1 {
		t.Errorf("number of containers after filtering = %d, want 1", got)
	}

	sendKey(tea.KeyPressMsg{Code: tea.KeyEsc})
	if got := countContainers(); got != 2 {
		t.Errorf("number of containers after clearing filter = %d, want 2", got)
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

func TestModel_ActionsRunning(t *testing.T) {
	mockCli := &mocks.MockContainerService{}
	m := New(mockCli)

	if m.IsActionRunning() {
		t.Error("IsActionRunning() should be false initially")
	}
	if m.isContainerActionRunning("c1") {
		t.Error("isContainerActionRunning(c1) should be false initially")
	}

	m.runningIDs["c1"] = true
	if !m.IsActionRunning() {
		t.Error("IsActionRunning() should be true when runningIDs has items")
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

func TestModel_RejectDuplicateRunCmdMsg(t *testing.T) {
	m := New(&mocks.MockContainerService{})
	dummyCmd := func() tea.Msg { return nil }

	var cmd tea.Cmd
	m, cmd = m.Update(RunCmdMsg{TargetID: "c1", Cmd: dummyCmd})
	if cmd == nil {
		t.Fatal("first RunCmdMsg should return a command")
	}

	// 1. Duplicate exact target
	_, cmd = m.Update(RunCmdMsg{TargetID: "c1", Cmd: dummyCmd})
	if cmd != nil {
		t.Errorf("duplicate RunCmdMsg for running target should return nil cmd, got %v", cmd)
	}

	// 2. Prune while c1 is running should be rejected
	_, cmd = m.Update(RunCmdMsg{TargetID: ActionTargetAll, Cmd: dummyCmd})
	if cmd != nil {
		t.Errorf("RunCmdMsg prune while c1 is running should return nil cmd, got %v", cmd)
	}

	// 3. Individual action while Prune is running should be rejected
	m2 := New(&mocks.MockContainerService{})
	m2, cmd = m2.Update(RunCmdMsg{TargetID: ActionTargetAll, Cmd: dummyCmd})
	if cmd == nil {
		t.Fatal("first prune RunCmdMsg should return a command")
	}
	_, cmd = m2.Update(RunCmdMsg{TargetID: "c1", Cmd: dummyCmd})
	if cmd != nil {
		t.Errorf("RunCmdMsg c1 while Prune is running should return nil cmd, got %v", cmd)
	}
}

func TestModel_ActionsBlockedWhilePruningOrRunning(t *testing.T) {
	mockCli := &mocks.MockContainerService{
		StreamContainerStatsFunc: func(ctx context.Context, id string) <-chan docker.ContainerMetrics {
			return make(chan docker.ContainerMetrics)
		},
	}
	m := New(mockCli)
	m.SetContainers([]docker.ContainerInfo{
		{ID: "c1", Name: "app1", State: docker.StateRunning},
	})
	m.focusZone = FocusLeftTree

	// 1. When ActionTargetAll is running (pruning)
	m.runningIDs[ActionTargetAll] = true
	_, cmd := m.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	if cmd != nil {
		t.Errorf("expected start key to be blocked while pruning, got %v", cmd)
	}
	_, cmd = m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	if cmd != nil {
		t.Errorf("expected restart key to be blocked while pruning, got %v", cmd)
	}
	_, cmd = m.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})
	if cmd != nil {
		t.Errorf("expected remove key to be blocked while pruning, got %v", cmd)
	}
	_, cmd = m.Update(tea.KeyPressMsg{Code: 'P', Text: "P"})
	if cmd != nil {
		t.Errorf("expected prune key to be blocked while pruning, got %v", cmd)
	}

	// 2. When container itself is running an action
	delete(m.runningIDs, ActionTargetAll)
	m.runningIDs["c1"] = true
	_, cmd = m.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	if cmd != nil {
		t.Errorf("expected start key to be blocked while container action is running, got %v", cmd)
	}
	_, cmd = m.Update(tea.KeyPressMsg{Code: 'P', Text: "P"})
	if cmd != nil {
		t.Errorf("expected prune key to be blocked while container c1 is running, got %v", cmd)
	}
}

func TestModel_ZoomToggle(t *testing.T) {
	mockCli := &mocks.MockContainerService{}
	m := New(mockCli)
	m.SetSize(100, 30)

	if m.IsZoomed() {
		t.Fatal("expected initially unzoomed")
	}
	origLeft := m.leftWidth()
	origRight := m.rightWidth()
	if origLeft == 0 || origRight == 0 {
		t.Fatalf("expected non-zero widths initially, got left=%d, right=%d", origLeft, origRight)
	}

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
