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
	case spinner.TickMsg:
		if m.action.IsRunning() {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
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

	case ImageActionMsg:
		m.action.Finish("")

	case tea.MouseClickMsg:
		if m.confirmModal.Active {
			return m, nil
		}
		leftWidth := (m.width * 52) / 100
		if msg.X < leftWidth {
			m.focusZone = FocusImagesList
			if targetIdx, ok := layout.ListClickTarget(
				msg.Y,
				m.height,
				m.cursorIndex,
				len(m.filteredIndices),
				m.searchBar.IsActive() || m.searchBar.HasFilter(),
			); ok {
				m.cursorIndex = targetIdx
				cmds = append(cmds, m.syncDetails())
			}
		} else {
			m.focusZone = FocusLayersPane
		}

	case tea.MouseWheelMsg:
		if m.confirmModal.Active {
			return m, nil
		}
		if m.focusZone == FocusImagesList {
			if msg.Button == tea.MouseWheelUp && m.cursorIndex > 0 {
				m.cursorIndex--
				cmds = append(cmds, m.syncDetails())
			} else if msg.Button == tea.MouseWheelDown && m.cursorIndex < len(m.filteredIndices)-1 {
				m.cursorIndex++
				cmds = append(cmds, m.syncDetails())
			}
		} else {
			var cmd tea.Cmd
			m.layersViewport, cmd = m.layersViewport.Update(msg)
			cmds = append(cmds, cmd)
		}

	case tea.KeyPressMsg:
		if m.confirmModal.Active {
			m.confirmModal, _ = m.confirmModal.Update(msg)
			if m.confirmModal.Confirmed {
				action := m.confirmModal.Action
				targetID := m.confirmModal.TargetID
				switch action {
				case modal.ActionRemove:
					m.action.Start(action, targetID, "", fmt.Sprintf("%s image...", action.Title()))
					return m, tea.Batch(m.removeImageCmd(targetID), m.spinner.Tick)
				case modal.ActionPrune:
					m.action.StartPrune("Pruning dangling images...")
					return m, tea.Batch(m.pruneImagesCmd(), m.spinner.Tick)
				}
			}
			return m, nil
		}

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
		case m.focusZone == FocusImagesList && !m.action.IsRunning() && key.Matches(msg, keys.SearchBarKeys.Activate):
			cmd := m.searchBar.Activate()
			return m, cmd

		case m.focusZone == FocusImagesList && key.Matches(msg, keys.SearchBarKeys.ClearFilter) && m.searchBar.HasFilter():
			m.searchBar.Clear()
			m.applyFilter()
			cmds = append(cmds, m.syncDetails())
			return m, tea.Batch(cmds...)

		case key.Matches(msg, keys.AppKeys.FocusToggle):
			if m.focusZone == FocusImagesList {
				m.focusZone = FocusLayersPane
			} else {
				m.focusZone = FocusImagesList
			}

		case key.Matches(msg, keys.ImageKeys.Remove):
			selected := m.SelectedImage()
			if selected != nil && !m.action.IsRunning() {
				targetDisplay := selected.Tag
				if targetDisplay == "<none>:<none>" {
					targetDisplay = docker.CleanImageID(selected.ID)
				}
				m.confirmModal.ShowImageAction(modal.ActionRemove, selected.ID, targetDisplay)
				return m, nil
			}

		case key.Matches(msg, keys.ImageKeys.Prune):
			if !m.action.IsRunning() {
				m.confirmModal.ShowImagePrune()
				return m, nil
			}

		default:
			if m.focusZone == FocusImagesList {
				switch {
				case key.Matches(msg, keys.ImageKeys.Up):
					if m.cursorIndex > 0 {
						m.cursorIndex--
						cmds = append(cmds, m.syncDetails())
					}
				case key.Matches(msg, keys.ImageKeys.Down):
					if m.cursorIndex < len(m.filteredIndices)-1 {
						m.cursorIndex++
						cmds = append(cmds, m.syncDetails())
					}
				}
			} else {
				switch {
				case key.Matches(msg, keys.ImageKeys.ToggleWrap):
					m.layersViewport.SoftWrap = !m.layersViewport.SoftWrap
					return m, nil
				case key.Matches(msg, keys.ImageKeys.Left):
					m.layersViewport.ScrollLeft(4)
					return m, nil
				case key.Matches(msg, keys.ImageKeys.Right):
					m.layersViewport.ScrollRight(4)
					return m, nil
				default:
					var cmd tea.Cmd
					m.layersViewport, cmd = m.layersViewport.Update(msg)
					cmds = append(cmds, cmd)
				}
			}
		}
	}

	return m, tea.Batch(cmds...)
}
