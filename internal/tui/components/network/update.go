package network

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
		// Reject if this target already has an action running
		if m.runningIDs[msg.TargetID] {
			return m, nil
		}
		// Reject prune if any individual network action is currently running
		if msg.TargetID == ActionTargetAll && len(m.runningIDs) > 0 {
			return m, nil
		}
		// Reject individual network action if prune is currently running
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

	case NetworkResultMsg:
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
		case m.focusZone == FocusNetworksList && !m.IsLoading() && key.Matches(msg, keys.SearchBarKeys.Activate):
			cmd := m.searchBar.Activate()
			return m, cmd

		case m.focusZone == FocusNetworksList && key.Matches(msg, keys.SearchBarKeys.ClearFilter) && m.searchBar.HasFilter():
			m.searchBar.Clear()
			m.applyFilter()
			m.updateViewportContent()
			return m, nil

		case key.Matches(msg, keys.AppKeys.ZoomToggle):
			m.isZoomed = !m.isZoomed
			m.updateViewportSizes()
			return m, nil

		case key.Matches(msg, keys.NetworkKeys.Focus, keys.AppKeys.FocusToggle):
			if m.focusZone == FocusNetworksList {
				m.focusZone = FocusDetailsPane
			} else {
				m.focusZone = FocusNetworksList
			}
			m.updateViewportSizes()
			return m, nil

		case key.Matches(msg, keys.NetworkKeys.Up):
			if m.focusZone == FocusNetworksList {
				if m.cursorIndex > 0 {
					m.cursorIndex--
					m.syncScroll()
					m.updateViewportContent()
				}
			} else {
				m.viewport.ScrollUp(1)
			}
			return m, nil

		case key.Matches(msg, keys.NetworkKeys.Down):
			if m.focusZone == FocusNetworksList {
				if m.cursorIndex < len(m.filteredIndices)-1 {
					m.cursorIndex++
					m.syncScroll()
					m.updateViewportContent()
				}
			} else {
				m.viewport.ScrollDown(1)
			}
			return m, nil

		case key.Matches(msg, keys.NetworkKeys.Left):
			if m.focusZone == FocusDetailsPane {
				m.viewport.ScrollLeft(4)
				return m, nil
			}

		case key.Matches(msg, keys.NetworkKeys.Right):
			if m.focusZone == FocusDetailsPane {
				m.viewport.ScrollRight(4)
				return m, nil
			}

		case key.Matches(msg, keys.NetworkKeys.ToggleWrap):
			if m.focusZone == FocusDetailsPane {
				m.viewport.SoftWrap = !m.viewport.SoftWrap
				return m, nil
			}

		case key.Matches(msg, keys.NetworkKeys.Delete):
			selected := m.SelectedNetwork()
			if selected != nil && !m.isNetworkActionRunning(selected.ID) {
				netID := selected.ID
				netName := selected.Name
				var question string
				if selected.ConnectedCount > 0 {
					question = fmt.Sprintf("Network '%s' has %d connected container(s)! Force remove?", netName, selected.ConnectedCount)
				} else {
					question = fmt.Sprintf("Are you sure you want to remove network %q?", netName)
				}
				cmdStr := "docker network rm " + netID
				return m, func() tea.Msg {
					return modal.ShowConfirmMsg{
						Question: question,
						Command:  cmdStr,
						Style:    modal.StyleDanger,
						OnConfirm: func() tea.Msg {
							return RunCmdMsg{
								TargetID: netID,
								Cmd:      m.removeNetworkCmd(netID),
							}
						},
					}
				}
			}

		case key.Matches(msg, keys.NetworkKeys.Prune):
			// Only permit prune when no network action is currently running
			if !m.IsLoading() {
				return m, func() tea.Msg {
					return modal.ShowConfirmMsg{
						Question: "Are you sure you want to remove all unused networks?",
						Command:  "docker network prune -f",
						Style:    modal.StyleWarning,
						OnConfirm: func() tea.Msg {
							return RunCmdMsg{
								TargetID: ActionTargetAll,
								Cmd:      m.pruneNetworksCmd(),
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
		leftWidth := m.leftWidth()
		isLeftClick := !m.isRightZoomed() && (m.isLeftZoomed() || msg.X < leftWidth)
		if isLeftClick {
			m.focusZone = FocusNetworksList
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
		if m.focusZone == FocusNetworksList {
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
