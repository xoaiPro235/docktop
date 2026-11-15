package container

import (
	"fmt"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/xoaiPro235/docktop/internal/docker"
	"github.com/xoaiPro235/docktop/internal/tui/components/searchbar"
	"github.com/xoaiPro235/docktop/internal/tui/keys"
	"github.com/xoaiPro235/docktop/internal/tui/layout"
	"github.com/xoaiPro235/docktop/internal/tui/modal"
	"github.com/xoaiPro235/docktop/internal/tui/theme"
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
			m.topPane.SetContent(msg.ContainerID, msg.Data, msg.Err)
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

	case ExecFinishedMsg, AttachFinishedMsg:
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

	case ComposeActionMsg:
		delete(m.runningActions, msg.TargetKey)
		m.rebuildTree()
		if msg.Err != nil {
			return m, func() tea.Msg {
				return modal.ShowErrorMsg{
					Title:   fmt.Sprintf("Compose %s Failed", msg.Action),
					Message: msg.Err.Error(),
				}
			}
		}
		return m, m.syncActivePane()

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
			if targetIdx, ok := layout.ListClickTarget(
				msg.Y,
				m.height,
				m.cursorIndex,
				m.scrollOffset,
				len(m.flatItems),
				m.searchBar.IsActive() || m.searchBar.HasFilter(),
			); ok {
				m.cursorIndex = targetIdx
				m.syncScroll()
				cmds = append(cmds, m.syncActivePane())
			}
		} else {
			m.focusZone = FocusRightPane
			selectedNode := m.SelectedNode()
			if selectedNode == nil || selectedNode.Type == NodeContainer {
				if msg.Y >= 1 && msg.Y <= 3 {
					relX := msg.X - leftWidth - 2
					subTabs := []struct {
						pane SubPane
						name string
					}{
						{SubPaneStats, "[Stats]"},
						{SubPaneLogs, "[Logs]"},
						{SubPaneTop, "[Top]"},
						{SubPaneInspect, "[Inspect]"},
					}

					currX := 0
					sepWidth := ansi.StringWidth(" | ")
					for _, tab := range subTabs {
						style := theme.Current.SubTabInactive
						if tab.pane == m.activePane {
							style = theme.Current.SubTabActive
						}
						w := ansi.StringWidth(style.Render(tab.name))
						if relX >= currX && relX < currX+w {
							m.activePane = tab.pane
							cmds = append(cmds, m.syncActivePane())
							break
						}
						currX += w + sepWidth
					}
				}
			} else if (selectedNode.Type == NodeProject || selectedNode.Type == NodeService) && selectedNode.Name != StandaloneProjectName {
				if msg.Y >= 1 && msg.Y <= 3 {
					relX := msg.X - leftWidth - 2
					composeTabs := []struct {
						pane ComposeSubPane
						name string
					}{
						{ComposeSubPaneConfig, "[Config]"},
						{ComposeSubPaneLogs, "[Logs]"},
					}

					currX := 0
					sepWidth := ansi.StringWidth(" | ")
					for _, tab := range composeTabs {
						style := theme.Current.SubTabInactive
						if tab.pane == m.composeActivePane {
							style = theme.Current.SubTabActive
						}
						w := ansi.StringWidth(style.Render(tab.name))
						if relX >= currX && relX < currX+w {
							m.composeActivePane = tab.pane
							cmds = append(cmds, m.syncActivePane())
							break
						}
						currX += w + sepWidth
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
				m.syncScroll()
				cmds = append(cmds, m.syncActivePane())
			} else if msg.Button == tea.MouseWheelDown && m.cursorIndex < len(m.flatItems)-1 {
				m.cursorIndex++
				m.syncScroll()
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

		if m.searchBar.IsActive() {
			var cmd tea.Cmd
			var evt searchbar.Event
			m.searchBar, cmd, evt = m.searchBar.Update(msg)
			if evt == searchbar.EventChanged || evt == searchbar.EventCanceled || evt == searchbar.EventSubmitted {
				m.rebuildTree()
				cmds = append(cmds, m.syncActivePane())
			}
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
		}

		// Global keys: Switch focus, toggle display options, cycle subpanes
		switch {
		case key.Matches(msg, keys.AppKeys.FocusToggle):
			if m.focusZone == FocusLeftTree {
				m.focusZone = FocusRightPane
			} else {
				m.focusZone = FocusLeftTree
			}
			return m, nil

		// Inspect pane: Toggle Raw / Formatted JSON
		case key.Matches(msg, keys.ContainerKeys.ToggleRaw):
			if m.activePane == SubPaneInspect {
				m.inspectPane.ToggleRaw()
				return m, nil
			}

		// Stats pane: Cycle Braille graph layout modes (Mirrored, Both Up, Both Down, Inverted)
		// Only active when not on a Compose node, so 'b' correctly triggers Compose Build on Project & Service nodes.
		case !m.SelectedNode().IsCompose() && key.Matches(msg, keys.ContainerKeys.ToggleBraille):
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
			case SubPaneInspect:
				m.inspectPane.ToggleWrap()
			case SubPaneTop:
				m.topPane.ToggleWrap()
			}
			return m, nil

		// Cycle subpanes: '[' and ']'
		case key.Matches(msg, keys.ContainerKeys.NextPane):
			selectedNode := m.SelectedNode()
			if selectedNode != nil && (selectedNode.Type == NodeProject || selectedNode.Type == NodeService) && selectedNode.Name != StandaloneProjectName {
				m.composeActivePane = (m.composeActivePane + 1) % 2
			} else {
				m.activePane = (m.activePane + 1) % 4
			}
			return m, m.syncActivePane()

		case key.Matches(msg, keys.ContainerKeys.PrevPane):
			selectedNode := m.SelectedNode()
			if selectedNode != nil && (selectedNode.Type == NodeProject || selectedNode.Type == NodeService) && selectedNode.Name != StandaloneProjectName {
				if m.composeActivePane == 0 {
					m.composeActivePane = ComposeSubPaneLogs
				} else {
					m.composeActivePane--
				}
			} else {
				if m.activePane == 0 {
					m.activePane = SubPaneInspect
				} else {
					m.activePane--
				}
			}
			return m, m.syncActivePane()
		}

		// -------------------------------------------------------------
		// 1. LEFT TREE FOCUSED: Navigation, Search, Compose & Container actions
		// -------------------------------------------------------------
		if m.focusZone == FocusLeftTree {
			selectedNode := m.SelectedNode()
			selectedContainer := m.SelectedContainer()

			switch {
			case key.Matches(msg, keys.SearchBarKeys.Activate):
				cmd := m.searchBar.Activate()
				return m, cmd

			case key.Matches(msg, keys.SearchBarKeys.ClearFilter) && m.searchBar.HasFilter():
				m.searchBar.Clear()
				m.rebuildTree()
				return m, m.syncActivePane()

			// --- Docker Compose lifecycle commands ---
			case selectedNode.IsCompose() && key.Matches(msg, keys.ComposeKeys.Up):
				opts := selectedNode.ComposeOpts()
				m.confirmModal.ShowComposeAction(modal.ActionComposeUp, selectedNode.Key, opts.ProjectName, opts.Service)
				return m, nil

			case selectedNode.IsCompose() && key.Matches(msg, keys.ComposeKeys.Build):
				opts := selectedNode.ComposeOpts()
				m.confirmModal.ShowComposeAction(modal.ActionComposeBuild, selectedNode.Key, opts.ProjectName, opts.Service)
				return m, nil

			case selectedNode.IsCompose() && key.Matches(msg, keys.ComposeKeys.Restart):
				opts := selectedNode.ComposeOpts()
				m.confirmModal.ShowComposeAction(modal.ActionComposeRestart, selectedNode.Key, opts.ProjectName, opts.Service)
				return m, nil

			case selectedNode.IsCompose() && key.Matches(msg, keys.ComposeKeys.Start):
				opts := selectedNode.ComposeOpts()
				if selectedNode.HasRunningContainers() {
					m.confirmModal.ShowComposeAction(modal.ActionComposeStop, selectedNode.Key, opts.ProjectName, opts.Service)
				} else {
					m.confirmModal.ShowComposeAction(modal.ActionComposeStart, selectedNode.Key, opts.ProjectName, opts.Service)
				}
				return m, nil

			case selectedNode.IsCompose() && key.Matches(msg, keys.ComposeKeys.Down):
				opts := selectedNode.ComposeOpts()
				switch selectedNode.Type {
				case NodeProject:
					m.confirmModal.ShowComposeAction(modal.ActionComposeDown, selectedNode.Key, opts.ProjectName, "")
				case NodeService:
					m.confirmModal.ShowComposeAction(modal.ActionComposeRm, selectedNode.Key, opts.ProjectName, opts.Service)
				}
				return m, nil

			case selectedNode.IsCompose() && key.Matches(msg, keys.ComposeKeys.Remove):
				opts := selectedNode.ComposeOpts()
				switch selectedNode.Type {
				case NodeService:
					m.confirmModal.ShowComposeAction(modal.ActionComposeRm, selectedNode.Key, opts.ProjectName, opts.Service)
					return m, nil
				case NodeProject:
					m.confirmModal.ShowComposeAction(modal.ActionComposeDown, selectedNode.Key, opts.ProjectName, "")
					return m, nil
				}

			// --- Container lifecycle commands ---
			case key.Matches(msg, keys.ContainerKeys.Exec):
				if selectedContainer != nil && selectedContainer.State == docker.StateRunning && !m.isContainerActionRunning(selectedContainer.ID) {
					cmd := m.dockerClient.ExecShellCmd(selectedContainer.ID)
					return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
						return ExecFinishedMsg{Err: err}
					})
				}

			case key.Matches(msg, keys.ContainerKeys.Attach):
				if selectedContainer != nil && selectedContainer.State == docker.StateRunning && !m.isContainerActionRunning(selectedContainer.ID) {
					cmd := m.dockerClient.AttachShellCmd(selectedContainer.ID)
					return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
						return AttachFinishedMsg{Err: err}
					})
				}

			case key.Matches(msg, keys.ContainerKeys.OpenBrowser):
				if selectedContainer != nil && selectedContainer.State == docker.StateRunning && !m.isContainerActionRunning(selectedContainer.ID) {
					if url, ok := selectedContainer.WebURL(); ok {
						if err := docker.OpenBrowser(url); err != nil {
							return m, func() tea.Msg {
								return modal.ShowErrorMsg{
									Title:   "Failed to Open Browser",
									Message: fmt.Sprintf("Could not launch default web browser for %s: %v", url, err),
								}
							}
						}
						return m, nil
					}
					return m, func() tea.Msg {
						return modal.ShowErrorMsg{
							Title:   "No Public Port",
							Message: fmt.Sprintf("Container %q has no ports mapped to the host.", selectedContainer.Name),
						}
					}
				}

			case key.Matches(msg, keys.ContainerKeys.Start):
				if selectedContainer != nil && !m.isContainerActionRunning(selectedContainer.ID) && !m.isContainerActionRunning(ActionTargetAll) {
					if selectedContainer.State == docker.StateRunning || selectedContainer.State == docker.StatePaused {
						m.confirmModal.ShowContainerAction(modal.ActionStop, selectedContainer.ID, selectedContainer.Name)
					} else {
						m.confirmModal.ShowContainerAction(modal.ActionStart, selectedContainer.ID, selectedContainer.Name)
					}
					return m, nil
				}

			case key.Matches(msg, keys.ContainerKeys.Pause):
				if selectedContainer != nil && !m.isContainerActionRunning(selectedContainer.ID) && !m.isContainerActionRunning(ActionTargetAll) {
					if selectedContainer.State == docker.StatePaused {
						m.confirmModal.ShowContainerAction(modal.ActionUnpause, selectedContainer.ID, selectedContainer.Name)
					} else {
						m.confirmModal.ShowContainerAction(modal.ActionPause, selectedContainer.ID, selectedContainer.Name)
					}
					return m, nil
				}

			case key.Matches(msg, keys.ContainerKeys.Restart):
				if selectedContainer != nil && !m.isContainerActionRunning(selectedContainer.ID) && !m.isContainerActionRunning(ActionTargetAll) {
					m.confirmModal.ShowContainerAction(modal.ActionRestart, selectedContainer.ID, selectedContainer.Name)
					return m, nil
				}

			case key.Matches(msg, keys.ContainerKeys.Remove):
				if selectedContainer != nil && !m.isContainerActionRunning(selectedContainer.ID) && !m.isContainerActionRunning(ActionTargetAll) {
					if selectedContainer.State == docker.StateRunning || selectedContainer.State == docker.StatePaused {
						m.confirmModal.ShowContainerAction(modal.ActionRemoveForce, selectedContainer.ID, selectedContainer.Name)
					} else {
						m.confirmModal.ShowContainerAction(modal.ActionRemove, selectedContainer.ID, selectedContainer.Name)
					}
					return m, nil
				}

			case key.Matches(msg, keys.ContainerKeys.Prune):
				if !m.IsActionRunning() {
					m.confirmModal.ShowContainerPrune()
					return m, nil
				}

			// --- Tree navigation ---
			case key.Matches(msg, keys.ContainerKeys.Up):
				if m.cursorIndex >= len(m.flatItems) {
					m.cursorIndex = max(0, len(m.flatItems)-1)
					m.syncScroll()
					cmds = append(cmds, m.syncActivePane())
				} else if m.cursorIndex > 0 {
					m.cursorIndex--
					m.syncScroll()
					cmds = append(cmds, m.syncActivePane())
				}

			case key.Matches(msg, keys.ContainerKeys.Down):
				if m.cursorIndex >= len(m.flatItems) {
					m.cursorIndex = max(0, len(m.flatItems)-1)
					m.syncScroll()
					cmds = append(cmds, m.syncActivePane())
				} else if m.cursorIndex < len(m.flatItems)-1 {
					m.cursorIndex++
					m.syncScroll()
					cmds = append(cmds, m.syncActivePane())
				}

			case key.Matches(msg, keys.ContainerKeys.ToggleExpand):
				if selectedNode != nil && len(selectedNode.Children) > 0 {
					selectedNode.Expanded = !selectedNode.Expanded
					m.expandedState[selectedNode.Key] = selectedNode.Expanded
					m.flatItems = Flatten(m.treeRoots, "")
					if m.cursorIndex >= len(m.flatItems) {
						m.cursorIndex = max(0, len(m.flatItems)-1)
					}
					cmds = append(cmds, m.syncActivePane())
				}

			case key.Matches(msg, keys.ContainerKeys.Collapse):
				if selectedNode != nil && len(selectedNode.Children) > 0 && selectedNode.Expanded {
					selectedNode.Expanded = false
					m.expandedState[selectedNode.Key] = false
					m.flatItems = Flatten(m.treeRoots, "")
					if m.cursorIndex >= len(m.flatItems) {
						m.cursorIndex = max(0, len(m.flatItems)-1)
					}
					cmds = append(cmds, m.syncActivePane())
				}

			case key.Matches(msg, keys.ContainerKeys.Expand):
				if selectedNode != nil && len(selectedNode.Children) > 0 && !selectedNode.Expanded {
					selectedNode.Expanded = true
					m.expandedState[selectedNode.Key] = true
					m.flatItems = Flatten(m.treeRoots, "")
					if m.cursorIndex >= len(m.flatItems) {
						m.cursorIndex = max(0, len(m.flatItems)-1)
					}
					cmds = append(cmds, m.syncActivePane())
				}
			}

			return m, tea.Batch(cmds...)
		}

		// -------------------------------------------------------------
		// 2. RIGHT PANE FOCUSED: Forward keys to active viewport
		// -------------------------------------------------------------
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

func (m Model) resolveComposeOpts(targetKey string) docker.ComposeOpts {
	var targetNode *TreeNode
	for _, item := range m.flatItems {
		if item.Node != nil && item.Node.Key == targetKey {
			targetNode = item.Node
			break
		}
	}
	if targetNode == nil {
		targetNode = m.SelectedNode()
	}
	if targetNode == nil {
		return docker.ComposeOpts{}
	}
	return targetNode.ComposeOpts()
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

	// Compose Actions
	case modal.ActionComposeUp:
		opts := m.resolveComposeOpts(targetID)
		return fmt.Sprintf("Starting %s...", targetName), m.composeUpCmd(targetID, targetName, opts)
	case modal.ActionComposeBuild:
		opts := m.resolveComposeOpts(targetID)
		return fmt.Sprintf("Building and starting %s...", targetName), m.composeBuildCmd(targetID, targetName, opts)
	case modal.ActionComposeRestart:
		opts := m.resolveComposeOpts(targetID)
		return fmt.Sprintf("Restarting %s...", targetName), m.composeRestartCmd(targetID, targetName, opts)
	case modal.ActionComposeStop:
		opts := m.resolveComposeOpts(targetID)
		return fmt.Sprintf("Stopping %s...", targetName), m.composeStopCmd(targetID, targetName, opts)
	case modal.ActionComposeStart:
		opts := m.resolveComposeOpts(targetID)
		return fmt.Sprintf("Starting %s...", targetName), m.composeStartCmd(targetID, targetName, opts)
	case modal.ActionComposeDown:
		opts := m.resolveComposeOpts(targetID)
		return fmt.Sprintf("Stopping and removing %s...", targetName), m.composeDownCmd(targetID, targetName, opts)
	case modal.ActionComposeRm:
		opts := m.resolveComposeOpts(targetID)
		return fmt.Sprintf("Removing %s...", targetName), m.composeRmCmd(targetID, targetName, opts)

	default:
		return "", nil
	}
}
