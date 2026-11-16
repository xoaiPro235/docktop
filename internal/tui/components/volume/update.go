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
	case RunCmdMsg:
		if m.runningIDs == nil {
			m.runningIDs = make(map[string]bool)
		}
		wasEmpty := len(m.runningIDs) == 0
		m.runningIDs[msg.TargetID] = true
		if wasEmpty {
			cmds = append(cmds, m.spinner.Tick)
		}
		cmds = append(cmds, msg.Cmd)
		return m, tea.Batch(cmds...)

	case VolumeResultMsg:
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

	case tea.KeyPressMsg:
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
		case m.focusZone == FocusVolumesList && !m.IsLoading() && key.Matches(msg, keys.SearchBarKeys.Activate):
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
			if selected != nil && !m.isVolumeActionRunning(selected.Name) {
				volName := selected.Name
				force := selected.InUse
				var question, cmdStr string
				if force {
					question = fmt.Sprintf("Volume is mounted by %d container(s)! Force remove %q?", len(selected.Containers), volName)
					cmdStr = fmt.Sprintf("docker volume rm -f %s", volName)
				} else {
					question = fmt.Sprintf("Are you sure you want to remove volume %q?", volName)
					cmdStr = fmt.Sprintf("docker volume rm %s", volName)
				}
				return m, func() tea.Msg {
					return modal.ShowConfirmMsg{
						Question: question,
						Command:  cmdStr,
						Style:    modal.StyleDanger,
						OnConfirm: func() tea.Msg {
							return RunCmdMsg{
								TargetID: volName,
								Cmd:      m.removeVolumeCmd(volName, force),
							}
						},
					}
				}
			}

		case key.Matches(msg, keys.VolumeKeys.Prune):
			if !m.isVolumeActionRunning(ActionTargetAll) {
				return m, func() tea.Msg {
					return modal.ShowConfirmMsg{
						Question: "Are you sure you want to remove all unused volumes?",
						Command:  "docker volume prune -f",
						Style:    modal.StyleWarning,
						OnConfirm: func() tea.Msg {
							return RunCmdMsg{
								TargetID: ActionTargetAll,
								Cmd:      m.pruneVolumesCmd(),
							}
						},
					}
				}
			}

		default:
			if m.focusZone == FocusDetailsPane {
				var cmd tea.Cmd
				m.viewport, cmd = m.viewport.Update(msg)
				cmds = append(cmds, cmd)
			}
		}

	case tea.MouseClickMsg:
		leftWidth := (m.width * 52) / 100
		if msg.X < leftWidth {
			m.focusZone = FocusVolumesList
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
				m.updateViewportContent()
			}
		} else {
			m.focusZone = FocusDetailsPane
		}

	case tea.MouseWheelMsg:
		if m.focusZone == FocusVolumesList {
			if msg.Button == tea.MouseWheelUp && m.cursorIndex > 0 {
				m.cursorIndex--
				m.syncScroll()
				m.updateViewportContent()
			} else if msg.Button == tea.MouseWheelDown && m.cursorIndex < len(m.filteredIndices)-1 {
				m.cursorIndex++
				m.syncScroll()
				m.updateViewportContent()
			}
		} else {
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}
