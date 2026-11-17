package image

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/xoaiPro235/docktop/internal/docker"
	"github.com/xoaiPro235/docktop/internal/tui/components/searchbar"
	"github.com/xoaiPro235/docktop/internal/tui/keys"
	"github.com/xoaiPro235/docktop/internal/tui/layout"
	"github.com/xoaiPro235/docktop/internal/tui/modal"
)

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case RunCmdMsg:
		if m.runningIDs == nil {
			m.runningIDs = make(map[string]bool)
		}
		// Reject if this target already has an action running
		if m.runningIDs[msg.TargetID] {
			return m, nil
		}
		// Reject prune if any individual image action is currently running
		if msg.TargetID == ActionTargetAll && len(m.runningIDs) > 0 {
			return m, nil
		}
		// Reject individual image action if prune is currently running
		if msg.TargetID != ActionTargetAll && m.runningIDs[ActionTargetAll] {
			return m, nil
		}
		wasEmpty := len(m.runningIDs) == 0
		m.runningIDs[msg.TargetID] = true
		if wasEmpty {
			cmds = append(cmds, m.spinner.Tick)
		}
		cmds = append(cmds, msg.Cmd)
		return m, tea.Batch(cmds...)

	case ImageResultMsg:
		delete(m.runningIDs, msg.TargetID)
		return m, nil

	case spinner.TickMsg:
		if len(m.runningIDs) > 0 {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	case ImageDetailsMsg:
		for i := range m.images {
			if m.images[i].ID == msg.ImageID {
				if msg.Details != nil {
					m.images[i].Details = msg.Details
				}
				if len(msg.Layers) > 0 {
					m.images[i].Layers = msg.Layers
				}
				break
			}
		}
		selected := m.SelectedImage()
		if selected != nil && selected.ID == msg.ImageID {
			m.updateViewportContent()
		}

	case tea.KeyPressMsg:
		if m.searchBar.IsActive() {
			var cmd tea.Cmd
			var evt searchbar.Event
			m.searchBar, cmd, evt = m.searchBar.Update(msg)
			if evt == searchbar.EventChanged || evt == searchbar.EventCanceled || evt == searchbar.EventSubmitted {
				m.applyFilter()
				cmds = append(cmds, m.syncDetails())
			}
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
		}

		switch {
		case m.focusZone == FocusImagesList && !m.IsLoading() && key.Matches(msg, keys.SearchBarKeys.Activate):
			cmd := m.searchBar.Activate()
			return m, cmd

		case m.focusZone == FocusImagesList && key.Matches(msg, keys.SearchBarKeys.ClearFilter) && m.searchBar.HasFilter():
			m.searchBar.Clear()
			m.applyFilter()
			cmds = append(cmds, m.syncDetails())
			return m, tea.Batch(cmds...)

		case key.Matches(msg, keys.AppKeys.ZoomToggle):
			m.isZoomed = !m.isZoomed
			m.updateViewportSizes()
			return m, nil

		case key.Matches(msg, keys.AppKeys.FocusToggle):
			if m.focusZone == FocusImagesList {
				m.focusZone = FocusLayersPane
			} else {
				m.focusZone = FocusImagesList
			}
			m.updateViewportSizes()
			return m, nil

		case key.Matches(msg, keys.ImageKeys.Up):
			if m.focusZone == FocusImagesList {
				if m.cursorIndex > 0 {
					m.cursorIndex--
					m.syncScroll()
					cmds = append(cmds, m.syncDetails())
				}
			} else {
				m.layersViewport.ScrollUp(1)
			}
			return m, tea.Batch(cmds...)

		case key.Matches(msg, keys.ImageKeys.Down):
			if m.focusZone == FocusImagesList {
				if m.cursorIndex < len(m.filteredIndices)-1 {
					m.cursorIndex++
					m.syncScroll()
					cmds = append(cmds, m.syncDetails())
				}
			} else {
				m.layersViewport.ScrollDown(1)
			}
			return m, tea.Batch(cmds...)

		case key.Matches(msg, keys.ImageKeys.Left):
			if m.focusZone == FocusLayersPane {
				m.layersViewport.ScrollLeft(4)
				return m, nil
			}

		case key.Matches(msg, keys.ImageKeys.Right):
			if m.focusZone == FocusLayersPane {
				m.layersViewport.ScrollRight(4)
				return m, nil
			}

		case key.Matches(msg, keys.ImageKeys.ToggleWrap):
			if m.focusZone == FocusLayersPane {
				m.layersViewport.SoftWrap = !m.layersViewport.SoftWrap
				return m, nil
			}

		case key.Matches(msg, keys.ImageKeys.Remove):
			selected := m.SelectedImage()
			if selected != nil && !m.isImageActionRunning(selected.ID) {
				targetDisplay := selected.Tag
				if targetDisplay == "<none>:<none>" {
					targetDisplay = docker.CleanImageID(selected.ID)
				}
				targetID := selected.ID
				return m, func() tea.Msg {
					return modal.ShowConfirmMsg{
						Question: fmt.Sprintf("Are you sure you want to remove image %q?", targetDisplay),
						Command:  fmt.Sprintf("docker rmi %s", targetID),
						Style:    modal.StyleDanger,
						OnConfirm: func() tea.Msg {
							return RunCmdMsg{TargetID: targetID, Cmd: m.removeImageCmd(targetID)}
						},
					}
				}
			}

		case key.Matches(msg, keys.ImageKeys.Prune):
			// Only permit prune when no image action is currently running
			if !m.IsLoading() {
				return m, func() tea.Msg {
					return modal.ShowConfirmMsg{
						Question: "Are you sure you want to remove all dangling images?",
						Command:  "docker image prune -f",
						Style:    modal.StyleWarning,
						OnConfirm: func() tea.Msg {
							return RunCmdMsg{TargetID: ActionTargetAll, Cmd: m.pruneImagesCmd()}
						},
					}
				}
			}

		default:
			if m.focusZone == FocusLayersPane {
				var cmd tea.Cmd
				m.layersViewport, cmd = m.layersViewport.Update(msg)
				cmds = append(cmds, cmd)
			}
		}

	case tea.MouseClickMsg:
		leftWidth := m.leftWidth()
		isLeftClick := !m.isRightZoomed() && (m.isLeftZoomed() || msg.X < leftWidth)
		if isLeftClick {
			m.focusZone = FocusImagesList
			if targetIdx, ok := layout.ListClickTarget(
				msg.Y,
				m.height,
				m.cursorIndex,
				m.scrollOffset,
				len(m.filteredIndices),
				m.searchBar.IsActive() || m.searchBar.HasFilter(),
			); ok {
				m.cursorIndex = targetIdx
				m.syncScroll()
				cmds = append(cmds, m.syncDetails())
			}
		} else {
			m.focusZone = FocusLayersPane
		}

	case tea.MouseWheelMsg:
		if m.focusZone == FocusImagesList {
			if msg.Button == tea.MouseWheelUp && m.cursorIndex > 0 {
				m.cursorIndex--
				m.syncScroll()
				cmds = append(cmds, m.syncDetails())
			} else if msg.Button == tea.MouseWheelDown && m.cursorIndex < len(m.filteredIndices)-1 {
				m.cursorIndex++
				m.syncScroll()
				cmds = append(cmds, m.syncDetails())
			}
		} else {
			var cmd tea.Cmd
			m.layersViewport, cmd = m.layersViewport.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}
