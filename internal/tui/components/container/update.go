package container

import (
	"fmt"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/xoaiPro235/docktop/internal/docker"
	"github.com/xoaiPro235/docktop/internal/tui/keys"
	"github.com/xoaiPro235/docktop/internal/tui/modal"
)

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case StatsResultMsg:
		if m.activePane == SubPaneStats && msg.ContainerID == m.selectedContainerID {
			rightWidth := m.width - (m.width * 35 / 100) - 1
			// graphWidth inside Render() = (rightWidth-2) - 4 = rightWidth-6
			// totalPointsNeeded inside renderBtopBraille = graphWidth * 2
			graphWidth := rightWidth - 6
			m.statsPane.AddCPUData(msg.Metrics.CPUPercent, graphWidth*2)
			m.statsPane.AddMemData(msg.Metrics.MemUsage, graphWidth*2)
			m.statsPane.SetMem(msg.Metrics.MemUsage, msg.Metrics.MemLimit, msg.Metrics.HasMemLimit)
			m.statsPane.SetIO(msg.Metrics.NetworkRx, msg.Metrics.NetworkTx, msg.Metrics.BlockRead, msg.Metrics.BlockWrite)
			if m.statsChan != nil {
				cmds = append(cmds, WaitForNextStat(m.statsChan, msg.ContainerID))
			}
		} else {
			// Message dropped (different container or pane)
			m.statsChan = nil
		}

	case StatsStreamClosedMsg:
		if msg.ContainerID == m.selectedContainerID {
			m.statsChan = nil
			m.statsCancel = nil
		}

	case LogBatchMsg:
		selectedNode := m.SelectedNode()
		if selectedNode != nil && (selectedNode.Type == NodeProject || selectedNode.Type == NodeService) {
			if m.composeActivePane == ComposeSubPaneLogs && msg.ContainerID == selectedNode.Key {
				m.logsPane.AppendLines(msg.Lines)
				if m.logChan != nil {
					cmds = append(cmds, WaitForNextLog(m.logChan, msg.ContainerID))
				}
			}
		} else if m.activePane == SubPaneLogs && m.selectedContainerID != "" && msg.ContainerID == m.selectedContainerID {
			m.logsPane.AppendLines(msg.Lines)
			if m.logChan != nil {
				cmds = append(cmds, WaitForNextLog(m.logChan, msg.ContainerID))
			}
		}

	case LogStreamClosedMsg:
		selectedNode := m.SelectedNode()
		if selectedNode != nil && (selectedNode.Type == NodeProject || selectedNode.Type == NodeService) {
			if msg.ContainerID == selectedNode.Key {
				m.logChan = nil
				m.logCancel = nil
			}
		} else if msg.ContainerID == m.selectedContainerID {
			m.logChan = nil
			m.logCancel = nil
		}

	case InspectLoadedMsg:
		if msg.Err == nil && msg.ContainerID == m.selectedContainerID {
			m.inspectPane.SetContent(msg.Data)
		}

	case TopLoadedMsg:
		if msg.ContainerID == m.selectedContainerID {
			if msg.Data != nil {
				m.topPane.SetContent(msg.ContainerID, msg.Data.Titles, msg.Data.Processes, msg.Err)
			} else {
				m.topPane.SetContent(msg.ContainerID, nil, nil, msg.Err)
			}
			// If still on SubPaneTop and container is running, schedule next poll after 3s
			selected := m.SelectedContainer()
			if m.activePane == SubPaneTop && selected != nil && selected.State == docker.StateRunning {
				cmds = append(cmds, m.pollTopAfter(3*time.Second, msg.ContainerID))
			}
		}

	case PollTopMsg:
		selected := m.SelectedContainer()
		if m.activePane == SubPaneTop && selected != nil && selected.ID == msg.ContainerID && selected.State == docker.StateRunning {
			cmds = append(cmds, m.fetchTopCmd(msg.ContainerID))
		}

	case ExecFinishedMsg:
		// Interactive terminal shell exited, resume TUI
		return m, m.syncActivePane()

	case spinner.TickMsg:
		if len(m.runningActions) > 0 {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	case ExecuteActionMsg:
		if m.runningActions == nil {
			m.runningActions = make(map[string]RunningAction)
		}
		wasEmpty := len(m.runningActions) == 0

		desc, cmd := m.executeAction(msg.Action, msg.TargetID, msg.TargetName)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		m.runningActions[msg.TargetID] = RunningAction{
			Action:      msg.Action,
			TargetName:  msg.TargetName,
			Description: desc,
		}

		if wasEmpty {
			cmds = append(cmds, m.spinner.Tick)
		}
		return m, tea.Batch(cmds...)

	case ContainerActionMsg:
		targetID := msg.ContainerID
		if msg.Action == modal.ActionPrune {
			targetID = ActionTargetAll 
		}
		delete(m.runningActions, targetID)

		if msg.Err == nil {
			switch msg.Action {
			case modal.ActionRemove, modal.ActionRemoveForce:
				var remaining []docker.ContainerInfo
				for _, c := range m.containers {
					if c.ID != msg.ContainerID {
						remaining = append(remaining, c)
					}
				}
				m.containers = remaining
			case modal.ActionPrune:
				var remaining []docker.ContainerInfo
				for _, c := range m.containers {
					if c.State == docker.StateRunning {
						remaining = append(remaining, c)
					}
				}
				m.containers = remaining
			default:
				for i := range m.containers {
					if m.containers[i].ID == msg.ContainerID {
						switch msg.Action {
						case modal.ActionPause:
							m.containers[i].State = docker.StatePaused
						case modal.ActionUnpause:
							m.containers[i].State = docker.StateRunning
						case modal.ActionStop:
							m.containers[i].State = docker.StateExited
						case modal.ActionStart, modal.ActionRestart:
							m.containers[i].State = docker.StateRunning
						}
						break
					}
				}
			}
			m.updateContainers(m.containers)
		}
		return m, m.syncActivePane()

	case tea.MouseClickMsg:
		if m.confirmModal.Active {
			return m, nil
		}
		leftWidth := max(m.width*35/100, 25)

		if msg.X <= leftWidth {
			m.focusZone = FocusLeftTree
			startIdx := 0
			maxVisibleItems := max(m.height-4, 1)
			if m.cursorIndex >= maxVisibleItems {
				startIdx = m.cursorIndex - maxVisibleItems + 1
			}

			// Tree rows begin below header (offset 3)
			offset := msg.Y - 3
			if offset >= 0 {
				targetIdx := startIdx + offset
				if targetIdx >= 0 && targetIdx < len(m.flatItems) {
					m.cursorIndex = targetIdx
					cmds = append(cmds, m.syncActivePane())
				}
			}
		} else {
			m.focusZone = FocusRightPane
			selectedNode := m.SelectedNode()
			if selectedNode == nil || selectedNode.Type == NodeContainer {
				if msg.Y >= 1 && msg.Y <= 3 {
					relX := msg.X - leftWidth - 2
					if relX >= 0 && relX < 9 {
						m.activePane = SubPaneStats
						cmds = append(cmds, m.syncActivePane())
					} else if relX >= 10 && relX < 18 {
						m.activePane = SubPaneLogs
						cmds = append(cmds, m.syncActivePane())
					} else if relX >= 19 && relX < 26 {
						m.activePane = SubPaneTop
						cmds = append(cmds, m.syncActivePane())
					} else if relX >= 27 && relX < 39 {
						m.activePane = SubPaneInspect
						cmds = append(cmds, m.syncActivePane())
					}
				}
			} else if (selectedNode.Type == NodeProject || selectedNode.Type == NodeService) && selectedNode.Name != StandaloneProjectName {
				if msg.Y >= 1 && msg.Y <= 3 {
					relX := msg.X - leftWidth - 2
					if relX >= 0 && relX < 10 {
						m.composeActivePane = ComposeSubPaneConfig
						cmds = append(cmds, m.syncActivePane())
					} else if relX >= 10 && relX < 19 {
						m.composeActivePane = ComposeSubPaneLogs
						cmds = append(cmds, m.syncActivePane())
					}
				}
			}
		}

	case tea.MouseWheelMsg:
		if m.confirmModal.Active {
			return m, nil
		}
		leftWidth := max(m.width*35/100, 25)

		if msg.X <= leftWidth {
			if msg.Button == tea.MouseWheelUp && m.cursorIndex > 0 {
				m.cursorIndex--
				cmds = append(cmds, m.syncActivePane())
			} else if msg.Button == tea.MouseWheelDown && m.cursorIndex < len(m.flatItems)-1 {
				m.cursorIndex++
				cmds = append(cmds, m.syncActivePane())
			}
		} else {
			var cmd tea.Cmd
			selectedNode := m.SelectedNode()
			if selectedNode != nil && (selectedNode.Type == NodeProject || selectedNode.Type == NodeService) {
				if m.composeActivePane == ComposeSubPaneLogs {
					m.logsPane, cmd = m.logsPane.Update(msg)
				} else {
					m.composePane, cmd = m.composePane.Update(msg)
				}
				cmds = append(cmds, cmd)
			} else {
				switch m.activePane {
				case SubPaneLogs:
					m.logsPane, cmd = m.logsPane.Update(msg)
					cmds = append(cmds, cmd)
				case SubPaneTop:
					m.topPane, cmd = m.topPane.Update(msg)
					cmds = append(cmds, cmd)
				case SubPaneInspect:
					m.inspectPane, cmd = m.inspectPane.Update(msg)
					cmds = append(cmds, cmd)
				}
			}
		}

	case tea.KeyPressMsg:
		if m.confirmModal.Active {
			m.confirmModal, _ = m.confirmModal.Update(msg)
			if m.confirmModal.Confirmed {
				return m.Update(ExecuteActionMsg{
					Action:     m.confirmModal.Action,
					TargetID:   m.confirmModal.TargetID,
					TargetName: m.confirmModal.TargetName,
				})
			}
			return m, nil
		}

		switch {
		// Switch focus Left / Right
		case key.Matches(msg, keys.AppKeys.FocusToggle):
			if m.focusZone == FocusLeftTree {
				m.focusZone = FocusRightPane
			} else {
				m.focusZone = FocusLeftTree
			}

		// Exec shell in running container (like lazydocker)
		case key.Matches(msg, keys.ContainerKeys.Exec):
			selected := m.SelectedContainer()
			if selected != nil && selected.State == docker.StateRunning && !m.isContainerActionRunning(selected.ID) {
				cmd := m.dockerClient.ExecShellCmd(selected.ID)
				return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
					return ExecFinishedMsg{Err: err}
				})
			}

		// Lifecycle: Start / Stop toggle with confirmation modal
		case key.Matches(msg, keys.ContainerKeys.StartStop):
			selected := m.SelectedContainer()
			if selected != nil && !m.isContainerActionRunning(selected.ID) && !m.isContainerActionRunning(ActionTargetAll) {
				if selected.State == docker.StateRunning || selected.State == docker.StatePaused {
					m.confirmModal.ShowContainerAction(modal.ActionStop, selected.ID, selected.Name)
				} else {
					m.confirmModal.ShowContainerAction(modal.ActionStart, selected.ID, selected.Name)
				}
				return m, nil
			}

		// Lifecycle: Pause / Unpause toggle with confirmation modal
		case key.Matches(msg, keys.ContainerKeys.Pause):
			selected := m.SelectedContainer()
			if selected != nil && !m.isContainerActionRunning(selected.ID) && !m.isContainerActionRunning(ActionTargetAll) {
				if selected.State == docker.StatePaused {
					m.confirmModal.ShowContainerAction(modal.ActionUnpause, selected.ID, selected.Name)
				} else {
					m.confirmModal.ShowContainerAction(modal.ActionPause, selected.ID, selected.Name)
				}
				return m, nil
			}

		// Lifecycle: Restart with confirmation modal
		case key.Matches(msg, keys.ContainerKeys.Restart):
			selected := m.SelectedContainer()
			if selected != nil && !m.isContainerActionRunning(selected.ID) && !m.isContainerActionRunning(ActionTargetAll) {
				m.confirmModal.ShowContainerAction(modal.ActionRestart, selected.ID, selected.Name)
				return m, nil
			}

		// Lifecycle: Remove container
		case key.Matches(msg, keys.ContainerKeys.Remove):
			selected := m.SelectedContainer()
			if selected != nil && !m.isContainerActionRunning(selected.ID) && !m.isContainerActionRunning(ActionTargetAll) {
				if selected.State == docker.StateRunning || selected.State == docker.StatePaused {
					m.confirmModal.ShowContainerAction(modal.ActionRemoveForce, selected.ID, selected.Name)
				} else {
					m.confirmModal.ShowContainerAction(modal.ActionRemove, selected.ID, selected.Name)
				}
				return m, nil
			}

		// Lifecycle: Prune stopped containers
		case key.Matches(msg, keys.ContainerKeys.Prune):
			if !m.IsActionRunning() {
				m.confirmModal.ShowContainerAction(modal.ActionPrune, ActionTargetAll, "all stopped containers")
				return m, nil
			}

		// Inspect pane: Toggle Raw / Formatted JSON
		case key.Matches(msg, keys.ContainerKeys.ToggleRaw):
			if m.activePane == SubPaneInspect {
				m.inspectPane.ToggleRaw()
				return m, nil
			}

		// Stats pane: Cycle Braille graph layout modes (Mirrored, Both Up, Both Down, Inverted)
		case key.Matches(msg, keys.ContainerKeys.ToggleBraille):
			if m.activePane == SubPaneStats {
				m.statsPane.CycleGraphMode()
				return m, nil
			}

		// Toggle text soft-wrap in right pane viewports (Logs, Top, Inspect, Compose)
		case key.Matches(msg, keys.ContainerKeys.ToggleWrap):
			selectedNode := m.SelectedNode()
			if selectedNode != nil && (selectedNode.Type == NodeProject || selectedNode.Type == NodeService) {
				if m.composeActivePane == ComposeSubPaneLogs {
					m.logsPane.ToggleWrap()
				} else {
					m.composePane.ToggleWrap()
				}
				return m, nil
			}
			switch m.activePane {
			case SubPaneLogs:
				m.logsPane.ToggleWrap()
				return m, nil
			case SubPaneInspect:
				m.inspectPane.ToggleWrap()
				return m, nil
			case SubPaneTop:
				m.topPane.ToggleWrap()
				return m, nil
			}

		// Cycle subpanes: '[' and ']'
		case key.Matches(msg, keys.ContainerKeys.NextPane):
			selectedNode := m.SelectedNode()
			if selectedNode != nil && (selectedNode.Type == NodeProject || selectedNode.Type == NodeService) && selectedNode.Name != StandaloneProjectName {
				m.composeActivePane = (m.composeActivePane + 1) % 2
				cmds = append(cmds, m.syncActivePane())
			} else {
				m.activePane = (m.activePane + 1) % 4
				cmds = append(cmds, m.syncActivePane())
			}
		case key.Matches(msg, keys.ContainerKeys.PrevPane):
			selectedNode := m.SelectedNode()
			if selectedNode != nil && (selectedNode.Type == NodeProject || selectedNode.Type == NodeService) && selectedNode.Name != StandaloneProjectName {
				if m.composeActivePane == 0 {
					m.composeActivePane = ComposeSubPaneLogs
				} else {
					m.composeActivePane--
				}
				cmds = append(cmds, m.syncActivePane())
			} else {
				if m.activePane == 0 {
					m.activePane = SubPaneInspect
				} else {
					m.activePane--
				}
				cmds = append(cmds, m.syncActivePane())
			}

		// Navigate tree (when left tree is focused)
		case m.focusZone == FocusLeftTree && key.Matches(msg, keys.ContainerKeys.Up):
			if m.cursorIndex > 0 {
				m.cursorIndex--
				cmds = append(cmds, m.syncActivePane())
			}
		case m.focusZone == FocusLeftTree && key.Matches(msg, keys.ContainerKeys.Down):
			if m.cursorIndex < len(m.flatItems)-1 {
				m.cursorIndex++
				cmds = append(cmds, m.syncActivePane())
			}

		// Space / Enter: Toggle expand/collapse
		case m.focusZone == FocusLeftTree && key.Matches(msg, keys.ContainerKeys.ToggleExpand):
			if len(m.flatItems) > 0 {
				selected := m.flatItems[m.cursorIndex].Node
				if len(selected.Children) > 0 {
					selected.Expanded = !selected.Expanded
					m.expandedState[selected.Key] = selected.Expanded
					m.flatItems = Flatten(m.treeRoots, "")
					if m.cursorIndex >= len(m.flatItems) {
						m.cursorIndex = max(0, len(m.flatItems)-1)
					}
					cmds = append(cmds, m.syncActivePane())
				}
			}

		// Left / h: Collapse open node
		case m.focusZone == FocusLeftTree && key.Matches(msg, keys.ContainerKeys.Collapse):
			if len(m.flatItems) > 0 {
				selected := m.flatItems[m.cursorIndex].Node
				if len(selected.Children) > 0 && selected.Expanded {
					selected.Expanded = false
					m.expandedState[selected.Key] = false
					m.flatItems = Flatten(m.treeRoots, "")
					if m.cursorIndex >= len(m.flatItems) {
						m.cursorIndex = max(0, len(m.flatItems)-1)
					}
					cmds = append(cmds, m.syncActivePane())
				}
			}

		// Right / l: Expand closed node
		case m.focusZone == FocusLeftTree && key.Matches(msg, keys.ContainerKeys.Expand):
			if len(m.flatItems) > 0 {
				selected := m.flatItems[m.cursorIndex].Node
				if len(selected.Children) > 0 && !selected.Expanded {
					selected.Expanded = true
					m.expandedState[selected.Key] = true
					m.flatItems = Flatten(m.treeRoots, "")
					cmds = append(cmds, m.syncActivePane())
				}
			}
		}

		// Forward key events to right pane when focused
		if m.focusZone == FocusRightPane {
			var cmd tea.Cmd
			selectedNode := m.SelectedNode()
			if selectedNode != nil && (selectedNode.Type == NodeProject || selectedNode.Type == NodeService) {
				if m.composeActivePane == ComposeSubPaneLogs {
					m.logsPane, cmd = m.logsPane.Update(msg)
					cmds = append(cmds, cmd)
				} else {
					m.composePane, cmd = m.composePane.Update(msg)
					cmds = append(cmds, cmd)
				}
			} else {
				switch m.activePane {
				case SubPaneLogs:
					m.logsPane, cmd = m.logsPane.Update(msg)
					cmds = append(cmds, cmd)
				case SubPaneTop:
					m.topPane, cmd = m.topPane.Update(msg)
					cmds = append(cmds, cmd)
				case SubPaneInspect:
					m.inspectPane, cmd = m.inspectPane.Update(msg)
					cmds = append(cmds, cmd)
				}
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m Model) executeAction(action modal.Action, targetID, targetName string) (string, tea.Cmd) {
	switch action {
	case modal.ActionStart:
		return fmt.Sprintf("Starting %s...", targetName), m.startContainerCmd(targetID)
	case modal.ActionStop:
		return fmt.Sprintf("Stopping %s...", targetName), m.stopContainerCmd(targetID)
	case modal.ActionRestart:
		return fmt.Sprintf("Restarting %s...", targetName), m.restartContainerCmd(targetID)
	case modal.ActionPause:
		return fmt.Sprintf("Pausing %s...", targetName), m.pauseContainerCmd(targetID)
	case modal.ActionUnpause:
		return fmt.Sprintf("Unpausing %s...", targetName), m.unpauseContainerCmd(targetID)
	case modal.ActionRemove:
		return fmt.Sprintf("Removing %s...", targetName), m.removeContainerCmd(targetID, false)
	case modal.ActionRemoveForce:
		return fmt.Sprintf("Force removing %s...", targetName), m.removeContainerCmd(targetID, true)
	case modal.ActionPrune:
		return "Pruning stopped containers...", m.pruneContainersCmd()
	default:
		return "", nil
	}
}
