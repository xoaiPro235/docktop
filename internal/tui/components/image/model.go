package image

import (
	"strings"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	mobyimage "github.com/moby/moby/api/types/image"
	"github.com/xoaiPro235/docktop/internal/docker"
	"github.com/xoaiPro235/docktop/internal/tui/components/searchbar"
	"github.com/xoaiPro235/docktop/internal/tui/modal"
	"github.com/xoaiPro235/docktop/internal/tui/theme"
)

type FocusZone int

const (
	FocusImagesList FocusZone = iota
	FocusLayersPane
)

type Model struct {
	dockerClient    docker.ImageService
	images          []docker.ImageInfo
	filteredIndices []int
	searchBar       searchbar.Model

	cursorIndex    int
	focusZone      FocusZone
	layersViewport viewport.Model
	viewportReady  bool
	activeImageID  string

	confirmModal modal.ConfirmModal
	spinner      spinner.Model
	action       modal.ActionState

	width  int
	height int
}

func New(cli docker.ImageService) Model {
	sp := spinner.New(
		spinner.WithSpinner(spinner.MiniDot),
		spinner.WithStyle(theme.Current.ActionLoading),
	)

	m := Model{
		dockerClient: cli,
		focusZone:    FocusImagesList,
		confirmModal: modal.NewConfirmModal(),
		spinner:      sp,
		searchBar:    searchbar.New("Filter images..."),
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
		m.layersViewport = viewport.New(viewport.WithWidth(innerRightWidth), viewport.WithHeight(innerRightHeight))
		m.layersViewport.SetHorizontalStep(4)
		m.viewportReady = true
	} else {
		m.layersViewport.SetWidth(innerRightWidth)
		m.layersViewport.SetHeight(innerRightHeight)
	}
	m.updateViewportContent()
}

func (m *Model) SetImages(images []docker.ImageInfo) tea.Cmd {
	// Preserve cached details and layers so switching tabs does not cause layers to flicker/jump
	cacheDetails := make(map[string]*mobyimage.InspectResponse)
	cacheLayers := make(map[string][]docker.ImageLayer)

	for i := range m.images {
		if m.images[i].Details != nil {
			cacheDetails[m.images[i].ID] = m.images[i].Details
		}
		if len(m.images[i].Layers) > 0 {
			cacheLayers[m.images[i].ID] = m.images[i].Layers
		}
	}

	for i := range images {
		if d, ok := cacheDetails[images[i].ID]; ok {
			images[i].Details = d
		}
		if l, ok := cacheLayers[images[i].ID]; ok {
			images[i].Layers = l
		}
	}

	var prevSelectedID string
	if sel := m.SelectedImage(); sel != nil {
		prevSelectedID = sel.ID
	}

	m.images = images
	m.applyFilter()
	if prevSelectedID != "" {
		found := false
		for i, idx := range m.filteredIndices {
			if m.images[idx].ID == prevSelectedID {
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

	selected := m.SelectedImage()
	if selected != nil && (selected.Details == nil || len(selected.Layers) == 0) {
		return m.fetchImageDetailsCmd(selected.ID)
	}
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

func (m *Model) SelectedImage() *docker.ImageInfo {
	if len(m.filteredIndices) > 0 && m.cursorIndex >= 0 && m.cursorIndex < len(m.filteredIndices) {
		actualIdx := m.filteredIndices[m.cursorIndex]
		if actualIdx >= 0 && actualIdx < len(m.images) {
			return &m.images[actualIdx]
		}
	}
	return nil
}

func (m *Model) applyFilter() {
	query := strings.ToLower(m.searchBar.Value())
	m.filteredIndices = nil

	if query == "" {
		m.filteredIndices = make([]int, len(m.images))
		for i := range m.images {
			m.filteredIndices[i] = i
		}
	} else {
		for i, img := range m.images {
			if strings.Contains(strings.ToLower(img.Tag), query) ||
				strings.Contains(strings.ToLower(img.ID), query) {
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
	selected := m.SelectedImage()
	if selected == nil {
		m.activeImageID = ""
		m.layersViewport.SetContent(theme.Current.MutedText.Render("No image selected."))
		return
	}

	content := FormatImageDetailsContent(selected, m.layersViewport.Width())
	prevOffset := m.layersViewport.YOffset()
	prevXOffset := m.layersViewport.XOffset()
	m.layersViewport.SetContent(content)

	if m.activeImageID == selected.ID {
		m.layersViewport.SetYOffset(prevOffset)
		m.layersViewport.SetXOffset(prevXOffset)
	} else {
		m.activeImageID = selected.ID
		m.layersViewport.GotoTop()
		m.layersViewport.SetXOffset(0)
	}
}

func (m *Model) syncDetails() tea.Cmd {
	m.layersViewport.GotoTop()
	m.layersViewport.SetXOffset(0)
	m.updateViewportContent()

	selected := m.SelectedImage()
	if selected != nil && (selected.Details == nil || len(selected.Layers) == 0) {
		return m.fetchImageDetailsCmd(selected.ID)
	}
	return nil
}
