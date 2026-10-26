package volume

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/xoaiPro235/docktop/internal/tui/components/searchbar"
	"github.com/xoaiPro235/docktop/internal/tui/keys"
	"github.com/xoaiPro235/docktop/internal/tui/layout"
	"github.com/xoaiPro235/docktop/internal/tui/modal"
)

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case VolumeActionMsg:
		statusMsg := ""
		if msg.Err == nil {
			statusMsg = fmt.Sprintf("Successfully executed %s!", msg.Action.Title())
		}
		m.action.Finish(statusMsg)
		return m, nil

	case spinner.TickMsg:
		if m.action.IsRunning() {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}

	case tea.KeyPressMsg:
		if m.confirmModal.Active {
			m.confirmModal, _ = m.confirmModal.Update(msg)
			if m.confirmModal.Confirmed {
				action := m.confirmModal.Action
				volName := m.confirmModal.TargetName
				switch action {
				case modal.ActionRemove:
					m.action.Start(action, volName, volName, fmt.Sprintf("Removing volume %s...", volName))
					return m, tea.Batch(m.removeVolumeCmd(volName, false), m.spinner.Tick)
				case modal.ActionRemoveForce:
					m.action.Start(action, volName, volName, fmt.Sprintf("Force removing volume %s...", volName))
					return m, tea.Batch(m.removeVolumeCmd(volName, true), m.spinner.Tick)
				case modal.ActionPrune:
					m.action.StartPrune("Pruning unused volumes...")
					return m, tea.Batch(m.pruneVolumesCmd(), m.spinner.Tick)
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
				m.updateViewportContent()
			}
			return m, cmd
		}

		switch {
		case m.focusZone == FocusVolumesList && !m.action.IsRunning() && key.Matches(msg, keys.SearchBarKeys.Activate):
			cmd := m.searchBar.Activate()
			return m, cmd

		case m.focusZone == FocusVolumesList && key.Matches(msg, keys.SearchBarKeys.ClearFilter) && m.searchBar.HasFilter():
			m.searchBar.Clear()
			m.applyFilter()
			m.updateViewportContent()
			return m, nil

		case key.Matches(msg, keys.VolumeKeys.Focus):
			if m.focusZone == FocusVolumesList {
				m.focusZone = FocusDetailsPane
			} else {
				m.focusZone = FocusVolumesList
			}
			return m, nil

		case key.Matches(msg, keys.VolumeKeys.Up):
			if m.focusZone == FocusVolumesList {
				if m.cursorIndex > 0 {
					m.cursorIndex--
					m.updateViewportContent()
				}
			} else {
				m.viewport.ScrollUp(1)
			}
			return m, nil

		case key.Matches(msg, keys.VolumeKeys.Down):
			if m.focusZone == FocusVolumesList {
				if m.cursorIndex < len(m.filteredIndices)-1 {
					m.cursorIndex++
					m.updateViewportContent()
				}
			} else {
				m.viewport.ScrollDown(1)
			}
			return m, nil

		case key.Matches(msg, keys.VolumeKeys.Left):
			if m.focusZone == FocusDetailsPane {
				m.viewport.ScrollLeft(4)
				return m, nil
			}

		case key.Matches(msg, keys.VolumeKeys.Right):
			if m.focusZone == FocusDetailsPane {
				m.viewport.ScrollRight(4)
				return m, nil
			}

		case key.Matches(msg, keys.VolumeKeys.ToggleWrap):
			if m.focusZone == FocusDetailsPane {
				m.viewport.SoftWrap = !m.viewport.SoftWrap
				return m, nil
			}

		case key.Matches(msg, keys.VolumeKeys.Remove):
			selected := m.SelectedVolume()
			if selected != nil && !m.action.IsRunning() {
				volName := selected.Name
				if selected.InUse {
					m.confirmModal.Show(
						fmt.Sprintf("Volume is mounted by %d container(s)! Force remove \"%s\"?", len(selected.Containers), volName),
						fmt.Sprintf("docker volume rm -f %s", volName),
						modal.StyleDanger,
						modal.ActionRemoveForce,
						volName,
						volName,
					)
				} else {
					m.confirmModal.ShowVolumeAction(modal.ActionRemove, volName)
				}
				return m, nil
			}

		case key.Matches(msg, keys.VolumeKeys.Prune):
			if !m.action.IsRunning() {
				m.confirmModal.ShowVolumePrune()
				return m, nil
			}

		default:
			if m.focusZone == FocusDetailsPane {
				var cmd tea.Cmd
				m.viewport, cmd = m.viewport.Update(msg)
				cmds = append(cmds, cmd)
			}
		}

	case tea.MouseClickMsg:
		if m.confirmModal.Active {
			return m, nil
		}

		leftWidth := (m.width * 52) / 100
		if msg.X < leftWidth {
			m.focusZone = FocusVolumesList
			if targetIdx, ok := layout.ListClickTarget(
				msg.Y,
				m.height,
				m.cursorIndex,
				len(m.filteredIndices),
				m.searchBar.IsActive() || m.searchBar.HasFilter(),
			); ok {
				m.cursorIndex = targetIdx
				m.updateViewportContent()
			}
		} else {
			m.focusZone = FocusDetailsPane
		}

	case tea.MouseWheelMsg:
		if m.confirmModal.Active {
			return m, nil
		}
		if m.focusZone == FocusDetailsPane {
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}
