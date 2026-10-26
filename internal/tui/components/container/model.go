package container

import (
	"context"
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/xoaiPro235/docktop/internal/docker"
	"github.com/xoaiPro235/docktop/internal/tui/components/searchbar"
	"github.com/xoaiPro235/docktop/internal/tui/modal"
	"github.com/xoaiPro235/docktop/internal/tui/theme"
)

type SubPane int

const (
	SubPaneStats SubPane = iota
	SubPaneLogs
	SubPaneTop
	SubPaneInspect
)

type FocusZone int

const (
	FocusLeftTree FocusZone = iota
	FocusRightPane
)

type ComposeSubPane int

const (
	ComposeSubPaneConfig ComposeSubPane = iota
	ComposeSubPaneLogs
)

// ActionTargetAll represents the special target identifier when pruning all stopped containers.
const ActionTargetAll = modal.TargetAll

type Model struct {
	dockerClient docker.ContainerService
	containers   []docker.ContainerInfo
	treeRoots    []*TreeNode
	flatItems    []FlatItem
	searchBar    searchbar.Model

	expandedState map[string]bool

	selectedContainerID   string
	selectedComposeTarget string
	logCancel             context.CancelFunc
	logChan               <-chan string
	statsCancel           context.CancelFunc
	statsChan             <-chan docker.ContainerMetrics

	cursorIndex       int
	focusZone         FocusZone
	activePane        SubPane
	composeActivePane ComposeSubPane

	statsPane   StatsPane
	logsPane    LogsPane
	topPane     TopPane
	inspectPane InspectPane
	composePane ComposePane

	confirmModal   modal.ConfirmModal
	spinner        spinner.Model
	runningActions map[string]RunningAction

	width  int
	height int
}

// RunningAction describes an ongoing asynchronous Docker operation on a container.
type RunningAction struct {
	Action      modal.Action
	TargetName  string
	Description string
}

func New(cli docker.ContainerService) Model {
	sp := spinner.New(
		spinner.WithSpinner(spinner.MiniDot),
		spinner.WithStyle(theme.Current.ActionLoading),
	)

	return Model{
		dockerClient:   cli,
		expandedState:  make(map[string]bool),
		runningActions: make(map[string]RunningAction),
		focusZone:      FocusLeftTree,
		activePane:     SubPaneStats,
		statsPane:      NewStatsPane(),
		logsPane:       NewLogsPane(),
		topPane:        NewTopPane(),
		inspectPane:    NewInspectPane(),
		composePane:    NewComposePane(),
		confirmModal:   modal.NewConfirmModal(),
		spinner:        sp,
		searchBar:      searchbar.New("Filter containers..."),
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m *Model) SelectedNode() *TreeNode {
	if len(m.flatItems) > 0 && m.cursorIndex >= 0 && m.cursorIndex < len(m.flatItems) {
		return m.flatItems[m.cursorIndex].Node
	}
	return nil
}

func (m *Model) SelectedContainer() *docker.ContainerInfo {
	node := m.SelectedNode()
	if node != nil {
		return node.Container
	}
	return nil
}

func (m *Model) rebuildTree() {
	var prevSelectedID string
	var prevName string

	if len(m.flatItems) > 0 && m.cursorIndex >= 0 && m.cursorIndex < len(m.flatItems) {
		selectedNode := m.flatItems[m.cursorIndex].Node
		prevName = selectedNode.Name
		if selectedNode.Container != nil {
			prevSelectedID = selectedNode.Container.ID
		}
	}

	query := strings.ToLower(m.searchBar.Value())
	if query == "" {
		m.treeRoots = BuildTree(m.containers, m.expandedState)
	} else {
		var matched []docker.ContainerInfo
		for _, c := range m.containers {
			if strings.Contains(strings.ToLower(c.Name), query) ||
				strings.Contains(strings.ToLower(c.Image), query) ||
				strings.Contains(strings.ToLower(c.Service), query) ||
				strings.Contains(strings.ToLower(c.ProjectName), query) ||
				strings.Contains(strings.ToLower(c.ID), query) {
				matched = append(matched, c)
			}
		}
		m.treeRoots = BuildTree(matched, nil)
	}
	m.flatItems = Flatten(m.treeRoots, "")

	if len(m.flatItems) == 0 {
		m.cursorIndex = 0
		return
	}

	// 1. Try to restore selection by Container ID
	if prevSelectedID != "" {
		for idx, item := range m.flatItems {
			if item.Node.Container != nil && item.Node.Container.ID == prevSelectedID {
				m.cursorIndex = idx
				return
			}
		}
	}

	// 2. Try to restore selection by Node Name (project/service)
	if prevName != "" {
		for idx, item := range m.flatItems {
			if item.Node.Name == prevName {
				m.cursorIndex = idx
				return
			}
		}
	}

	// 3. Fallback: select first container if available
	for idx, item := range m.flatItems {
		if item.Node.Container != nil {
			m.cursorIndex = idx
			return
		}
	}

	// 4. Fallback: clamp cursor if out of bounds
	if m.cursorIndex >= len(m.flatItems) {
		m.cursorIndex = max(0, len(m.flatItems)-1)
	}
}

func (m *Model) updateContainers(newContainers []docker.ContainerInfo) {
	m.containers = newContainers
	m.rebuildTree()
}

func (m *Model) SetContainers(list []docker.ContainerInfo) tea.Cmd {
	m.updateContainers(list)
	return m.syncActivePane()
}

func (m *Model) IsSearching() bool {
	return m.searchBar.IsActive()
}

func (m *Model) stopLogStream() {
	if m.logCancel != nil {
		m.logCancel()
		m.logCancel = nil
		m.logChan = nil
	}
}

func (m *Model) stopStatsStream() {
	if m.statsCancel != nil {
		m.statsCancel()
		m.statsCancel = nil
		m.statsChan = nil
	}
}

// Pause stops active log and stats streaming
func (m *Model) Pause() {
	m.stopLogStream()
	m.stopStatsStream()
}

// Resume synchronizes the active pane when returning to this tab
func (m *Model) Resume() tea.Cmd {
	return m.syncActivePane()
}

func (m *Model) IsModalActive() bool {
	return m.confirmModal.Active
}

func (m *Model) isContainerActionRunning(id string) bool {
	if m.runningActions == nil {
		return false
	}
	_, ok := m.runningActions[id]
	return ok
}

func (m *Model) IsActionRunning() bool {
	return len(m.runningActions) > 0
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h

	leftWidth := max(w*35/100, 25)
	rightWidth := max(w-leftWidth-1, 20)

	// Inner dimensions subtracting borders
	innerRightWidth := rightWidth - 2
	innerRightHeight := h - 2
	// Subtract header tab row and footer info line
	viewportHeight := max(innerRightHeight-2, 1)

	m.logsPane.SetSize(innerRightWidth, viewportHeight)
	m.inspectPane.SetSize(innerRightWidth, viewportHeight)
	m.topPane.SetSize(innerRightWidth, viewportHeight)
	m.composePane.SetSize(innerRightWidth, viewportHeight)
}

func (m *Model) syncComposeNode(node *TreeNode) tea.Cmd {
	m.selectedContainerID = ""
	m.stopStatsStream()

	if node.Name == StandaloneProjectName || m.composeActivePane == ComposeSubPaneConfig {
		m.stopLogStream()
		m.composePane.LoadProject(node)
		return nil
	}

	// ComposeSubPaneLogs
	if m.selectedComposeTarget != node.Key {
		m.selectedComposeTarget = node.Key
		m.stopLogStream()
		m.logsPane.Reset()
	}
	if m.logChan == nil {
		return m.startComposeLogStream(node)
	}
	return nil
}

func (m *Model) syncActivePane() tea.Cmd {
	selectedNode := m.SelectedNode()
	if selectedNode != nil && (selectedNode.Type == NodeProject || selectedNode.Type == NodeService) {
		return m.syncComposeNode(selectedNode)
	}

	m.selectedComposeTarget = ""

	selected := m.SelectedContainer()
	var currentID string
	if selected != nil {
		currentID = selected.ID
	}

	var cmds []tea.Cmd

	if currentID != m.selectedContainerID {
		m.selectedContainerID = currentID
		m.statsPane.Reset()
		m.Pause()
	}

	switch m.activePane {
	case SubPaneStats:
		m.stopLogStream()
		if currentID != "" && m.statsChan == nil && selected != nil && selected.State == docker.StateRunning {
			cmds = append(cmds, m.startStatsStream(currentID))
		} else if currentID == "" || selected == nil || selected.State != docker.StateRunning {
			m.stopStatsStream()
			m.statsPane.Reset()
		}
	case SubPaneLogs:
		m.stopStatsStream()
		if currentID != "" && m.logChan == nil {
			cmds = append(cmds, m.startLogStream(currentID))
		} else if currentID == "" {
			m.stopLogStream()
			m.logsPane.Reset()
		}
	case SubPaneTop:
		m.Pause()
		if currentID != "" {
			m.topPane.SetLoading(currentID)
			cmds = append(cmds, m.fetchTopCmd(currentID))
		} else {
			m.topPane.Reset()
		}
	case SubPaneInspect:
		m.Pause()
		if currentID != "" {
			cmds = append(cmds, m.fetchInspectCmd(currentID))
		} else {
			m.inspectPane.Reset()
		}
	}

	return tea.Batch(cmds...)
}
