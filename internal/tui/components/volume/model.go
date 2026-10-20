package volume

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
	FocusVolumesList FocusZone = iota
	FocusDetailsPane
)

type Model struct {
	dockerClient docker.VolumeService
	volumes      []docker.VolumeInfo

	cursorIndex      int
	focusZone        FocusZone
	viewport         viewport.Model
	viewportReady    bool
	activeVolumeName string

	confirmModal modal.ConfirmModal
	spinner      spinner.Model
	action       modal.ActionState

	width  int
	height int
}

func New(cli docker.VolumeService) Model {
	sp := spinner.New(
		spinner.WithSpinner(spinner.MiniDot),
		spinner.WithStyle(theme.Current.ActionLoading),
	)

	return Model{
		dockerClient: cli,
		focusZone:    FocusVolumesList,
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

func (m *Model) SetVolumes(volumes []docker.VolumeInfo) tea.Cmd {
	var prevSelectedName string
	if m.cursorIndex >= 0 && m.cursorIndex < len(m.volumes) {
		prevSelectedName = m.volumes[m.cursorIndex].Name
	}

	m.volumes = volumes
	if prevSelectedName != "" {
		found := false
		for i, v := range m.volumes {
			if v.Name == prevSelectedName {
				m.cursorIndex = i
				found = true
				break
			}
		}
		if !found && m.cursorIndex >= len(m.volumes) {
			m.cursorIndex = max(0, len(m.volumes)-1)
		}
	} else if m.cursorIndex >= len(m.volumes) {
		m.cursorIndex = max(0, len(m.volumes)-1)
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

func (m *Model) SelectedVolume() *docker.VolumeInfo {
	if len(m.volumes) > 0 && m.cursorIndex >= 0 && m.cursorIndex < len(m.volumes) {
		return &m.volumes[m.cursorIndex]
	}
	return nil
}

func (m *Model) updateViewportContent() {
	if !m.viewportReady {
		return
	}
	selected := m.SelectedVolume()
	if selected == nil {
		m.activeVolumeName = ""
		m.viewport.SetContent(theme.Current.MutedText.Render("No volume selected."))
		return
	}

	prevOffset := m.viewport.YOffset()
	prevXOffset := m.viewport.XOffset()
	m.viewport.SetContent(FormatVolumeViewportContent(selected, m.viewport.Width()))

	if m.activeVolumeName == selected.Name {
		m.viewport.SetYOffset(prevOffset)
		m.viewport.SetXOffset(prevXOffset)
	} else {
		m.activeVolumeName = selected.Name
		m.viewport.GotoTop()
		m.viewport.SetXOffset(0)
	}
}
