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
	"github.com/xoaiPro235/docktop/internal/tui/theme"
)

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case NetworkActionMsg:
		statusMsg := ""
		if msg.Err == nil {
			statusMsg = theme.Current.StatusRunning.Render(fmt.Sprintf("Successfully executed network %s!", msg.Action.Title()))
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
				targetID := m.confirmModal.TargetID
				targetName := m.confirmModal.TargetName
				switch action {
				case modal.ActionDelete, modal.ActionRemove:
					m.action.Start(action, targetID, targetName, fmt.Sprintf("Deleting network %s...", targetName))
					return m, tea.Batch(m.removeNetworkCmd(targetID), m.spinner.Tick)
				case modal.ActionPrune:
					m.action.StartPrune("Pruning unused networks...")
					return m, tea.Batch(m.pruneNetworksCmd(), m.spinner.Tick)
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
		case m.focusZone == FocusNetworksList && !m.action.IsRunning() && key.Matches(msg, keys.SearchBarKeys.Activate):
			cmd := m.searchBar.Activate()
			return m, cmd

		case m.focusZone == FocusNetworksList && key.Matches(msg, keys.SearchBarKeys.ClearFilter) && m.searchBar.HasFilter():
			m.searchBar.Clear()
			m.applyFilter()
			m.updateViewportContent()
			return m, nil

		case key.Matches(msg, keys.NetworkKeys.Focus):
			if m.focusZone == FocusNetworksList {
				m.focusZone = FocusDetailsPane
			} else {
				m.focusZone = FocusNetworksList
			}
			return m, nil

		case key.Matches(msg, keys.NetworkKeys.Up):
			if m.focusZone == FocusNetworksList {
				if m.cursorIndex > 0 {
					m.cursorIndex--
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
			if selected != nil && !m.action.IsRunning() {
				if selected.IsSystem {
					return m, func() tea.Msg {
						return modal.ShowErrorMsg{
							Title:   "Delete Network Failed",
							Message: fmt.Sprintf("Cannot delete default system network '%s'", selected.Name),
						}
					}
				}

				netID := selected.ID
				netName := selected.Name
				if selected.ConnectedCount > 0 {
					m.confirmModal.Show(
						fmt.Sprintf("Network '%s' has %d connected container(s)! Force remove?", netName, selected.ConnectedCount),
						fmt.Sprintf("docker network rm %s", netName),
						modal.StyleDanger,
						modal.ActionDelete,
						netID,
						netName,
					)
				} else {
					m.confirmModal.ShowNetworkAction(modal.ActionDelete, netID, netName)
				}
				return m, nil
			}

		case key.Matches(msg, keys.NetworkKeys.Prune):
			if !m.action.IsRunning() {
				m.confirmModal.ShowNetworkPrune()
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
			m.focusZone = FocusNetworksList
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
