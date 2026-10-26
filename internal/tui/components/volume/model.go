package volume

import (
	"strings"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/xoaiPro235/docktop/internal/docker"
	"github.com/xoaiPro235/docktop/internal/tui/components/searchbar"
	"github.com/xoaiPro235/docktop/internal/tui/modal"
	"github.com/xoaiPro235/docktop/internal/tui/theme"
)

type FocusZone int

const (
	FocusVolumesList FocusZone = iota
	FocusDetailsPane
)

type Model struct {
	dockerClient    docker.VolumeService
	volumes         []docker.VolumeInfo
	filteredIndices []int
	searchBar       searchbar.Model

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

	m := Model{
		dockerClient: cli,
		focusZone:    FocusVolumesList,
		confirmModal: modal.NewConfirmModal(),
		spinner:      sp,
		searchBar:    searchbar.New("Filter volumes..."),
	}
	m.applyFilter()
	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m *Model) SetSize(w, h int) {
	m.width = w
	m.height = h

	leftWidth := (w * 52) / 100
	rightWidth := w - leftWidth
	innerRightWidth := max(rightWidth-2, 10)
	innerRightHeight := max(h-2, 1)

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
	if sel := m.SelectedVolume(); sel != nil {
		prevSelectedName = sel.Name
	}

	m.volumes = volumes
	m.applyFilter()
	if prevSelectedName != "" {
		found := false
		for i, idx := range m.filteredIndices {
			if m.volumes[idx].Name == prevSelectedName {
				m.cursorIndex = i
				found = true
				break
			}
		}
		if !found && m.cursorIndex >= len(m.filteredIndices) {
			m.cursorIndex = max(0, len(m.filteredIndices)-1)
		}
	} else if m.cursorIndex >= len(m.filteredIndices) {
		m.cursorIndex = max(0, len(m.filteredIndices)-1)
	}

	m.updateViewportContent()
	return nil
}

func (m *Model) IsModalActive() bool {
	return m.confirmModal.Active
}

func (m *Model) IsSearching() bool {
	return m.searchBar.IsActive()
}

func (m *Model) IsActionRunning() bool {
	return m.action.IsRunning()
}

func (m *Model) SelectedVolume() *docker.VolumeInfo {
	if len(m.filteredIndices) > 0 && m.cursorIndex >= 0 && m.cursorIndex < len(m.filteredIndices) {
		actualIdx := m.filteredIndices[m.cursorIndex]
		if actualIdx >= 0 && actualIdx < len(m.volumes) {
			return &m.volumes[actualIdx]
		}
	}
	return nil
}

func (m *Model) applyFilter() {
	query := strings.ToLower(m.searchBar.Value())
	m.filteredIndices = nil

	if query == "" {
		m.filteredIndices = make([]int, len(m.volumes))
		for i := range m.volumes {
			m.filteredIndices[i] = i
		}
	} else {
		for i, v := range m.volumes {
			if strings.Contains(strings.ToLower(v.Name), query) ||
				strings.Contains(strings.ToLower(v.Driver), query) {
				m.filteredIndices = append(m.filteredIndices, i)
			}
		}
	}

	if m.cursorIndex >= len(m.filteredIndices) {
		m.cursorIndex = max(0, len(m.filteredIndices)-1)
	}
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
