package network

import (
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/xoaiPro235/docktop/internal/docker"
	"github.com/xoaiPro235/docktop/internal/tui/modal"
	"github.com/xoaiPro235/docktop/internal/tui/theme"
)

type FocusZone int

const (
	FocusNetworksList FocusZone = iota
	FocusDetailsPane
)

type Model struct {
	dockerClient docker.NetworkService
	networks     []docker.NetworkInfo

	cursorIndex     int
	focusZone       FocusZone
	viewport        viewport.Model
	viewportReady   bool
	activeNetworkID string

	confirmModal modal.ConfirmModal
	spinner      spinner.Model
	action       modal.ActionState

	width  int
	height int
}

func New(cli docker.NetworkService) Model {
	sp := spinner.New(
		spinner.WithSpinner(spinner.MiniDot),
		spinner.WithStyle(theme.Current.ActionLoading),
	)

	return Model{
		dockerClient: cli,
		focusZone:    FocusNetworksList,
		confirmModal: modal.NewConfirmModal(),
		spinner:      sp,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h

	leftWidth := (w * 52) / 100
	rightWidth := w - leftWidth
	innerRightWidth := rightWidth - 2
	if innerRightWidth < 10 {
		innerRightWidth = 10
	}
	innerRightHeight := h - 2
	if innerRightHeight < 1 {
		innerRightHeight = 1
	}

	if !m.viewportReady {
		m.viewport = viewport.New(viewport.WithWidth(innerRightWidth), viewport.WithHeight(innerRightHeight))
		m.viewport.SetHorizontalStep(4)
		m.viewportReady = true
	} else {
		m.viewport.SetWidth(innerRightWidth)
		m.viewport.SetHeight(innerRightHeight)
	}
	m.updateViewportContent()
}

func (m *Model) SetNetworks(networks []docker.NetworkInfo) tea.Cmd {
	var prevSelectedID string
	if m.cursorIndex >= 0 && m.cursorIndex < len(m.networks) {
		prevSelectedID = m.networks[m.cursorIndex].ID
	}

	m.networks = networks
	if prevSelectedID != "" {
		found := false
		for i, n := range m.networks {
			if n.ID == prevSelectedID {
				m.cursorIndex = i
				found = true
				break
			}
		}
		if !found && m.cursorIndex >= len(m.networks) {
			m.cursorIndex = max(0, len(m.networks)-1)
		}
	} else if m.cursorIndex >= len(m.networks) {
		m.cursorIndex = max(0, len(m.networks)-1)
	}

	m.updateViewportContent()
	return nil
}

func (m *Model) IsModalActive() bool {
	return m.confirmModal.Active
}

func (m *Model) IsActionRunning() bool {
	return m.action.IsRunning()
}

func (m *Model) SelectedNetwork() *docker.NetworkInfo {
	if len(m.networks) > 0 && m.cursorIndex >= 0 && m.cursorIndex < len(m.networks) {
		return &m.networks[m.cursorIndex]
	}
	return nil
}

func (m *Model) updateViewportContent() {
	if !m.viewportReady {
		return
	}
	selected := m.SelectedNetwork()
	if selected == nil {
		m.activeNetworkID = ""
		m.viewport.SetContent(theme.Current.MutedText.Render("No network selected."))
		return
	}

	prevOffset := m.viewport.YOffset()
	prevXOffset := m.viewport.XOffset()
	m.viewport.SetContent(FormatNetworkViewportContent(selected, m.viewport.Width()))

	if m.activeNetworkID == selected.ID {
		m.viewport.SetYOffset(prevOffset)
		m.viewport.SetXOffset(prevXOffset)
	} else {
		m.activeNetworkID = selected.ID
		m.viewport.GotoTop()
		m.viewport.SetXOffset(0)
	}
}
