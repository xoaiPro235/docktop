package container

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/xoaiPro235/docktop/internal/docker"
	"github.com/xoaiPro235/docktop/internal/tui/keys"
	"github.com/xoaiPro235/docktop/internal/tui/layout"
	"github.com/xoaiPro235/docktop/internal/tui/theme"
)

func (m Model) View() string {
	leftWidth := max(m.width*35/100, 25)
	rightWidth := max(m.width-leftWidth-1, 20)

	leftBorderColor := theme.Current.Palette.BorderNormal
	rightBorderColor := theme.Current.Palette.BorderNormal
	if m.focusZone == FocusLeftTree {
		leftBorderColor = theme.Current.Palette.BorderFocused
	} else {
		rightBorderColor = theme.Current.Palette.BorderFocused
	}

	leftStyle := lipgloss.NewStyle().
		Width(leftWidth).
		Height(m.height).
		MaxHeight(m.height).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(leftBorderColor)

	rightStyle := lipgloss.NewStyle().
		Width(rightWidth).
		Height(m.height).
		MaxHeight(m.height).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(rightBorderColor)

	// 1. Render tree on left (windowing scrolls with cursor)
	var leftContent strings.Builder
	innerLeftWidth := max(leftWidth-2, 1)

	if m.searchBar.IsActive() || m.searchBar.HasFilter() {
		leftContent.WriteString(m.searchBar.View(innerLeftWidth))
		leftContent.WriteString("\n\n")
	}

	headerTitle := ansi.Truncate("PROJECT / SERVICE / CONTAINER", innerLeftWidth, "…")
	leftContent.WriteString(theme.Current.TreeHeader.Render(headerTitle))
	leftContent.WriteString("\n\n")

	if len(m.flatItems) == 0 {
		if m.searchBar.HasFilter() {
			leftContent.WriteString("  ")
			leftContent.WriteString(theme.Current.MutedText.Render(fmt.Sprintf("No containers matching %q", m.searchBar.Value())))
			leftContent.WriteString("\n")
		} else {
			leftContent.WriteString(theme.Current.MutedText.Padding(1).Render("No containers found."))
			leftContent.WriteString("\n")
		}
	} else {
		startIdx, endIdx, _ := layout.ListWindow(
			m.height,
			m.cursorIndex,
			len(m.flatItems),
			m.searchBar.IsActive() || m.searchBar.HasFilter(),
		)

		for i := startIdx; i < endIdx; i++ {
			item := m.flatItems[i]

			var cursorMark string
			if i == m.cursorIndex {
				cursorMark = "> "
			} else {
				cursorMark = "  "
			}

			branch := theme.Current.TreeBranch.Render(item.Prefix)
			title := item.Node.DisplayTitle()

			var styledTitle string
			switch item.Node.Type {
			case NodeProject:
				styledTitle = theme.Current.TreeProject.Render(title)
			case NodeService:
				styledTitle = theme.Current.TreeService.Render(title)
			case NodeContainer:
				if item.Node.Container != nil {
					_, isSelfAction := m.runningActions[item.Node.Container.ID]
					isPruningThis := m.isContainerActionRunning(ActionTargetAll) && item.Node.Container.State != docker.StateRunning

					if isSelfAction || isPruningThis {
						styledTitle = theme.Current.TreeSelected.Render(m.spinner.View() + " " + title)
					} else if item.Node.Container.State == docker.StateRunning {
						styledTitle = theme.Current.TreeContainerRunning.Render(title)
					} else {
						styledTitle = theme.Current.TreeContainerStopped.Render(title)
					}
				}
			default:
				styledTitle = title
			}

			line := fmt.Sprintf("%s%s%s", cursorMark, branch, styledTitle)
			if i == m.cursorIndex && m.focusZone == FocusLeftTree {
				line = theme.Current.TreeSelected.Render(line)
			}
			leftContent.WriteString(line)
			leftContent.WriteString("\n")
		}
	}

	// 2. Render Right Pane
	selectedNode := m.SelectedNode()
	var rightContent string

	if selectedNode != nil && (selectedNode.Type == NodeProject || selectedNode.Type == NodeService) {
		if selectedNode.Name == StandaloneProjectName {
			headerBar := theme.Current.TreeProject.Render("── STANDALONE CONTAINERS ──")
			rightContent = headerBar + "\n" + m.composePane.Render()
		} else {
			subTabs := []string{"[Config]", "[Logs]"}
			for i := range subTabs {
				if ComposeSubPane(i) == m.composeActivePane {
					subTabs[i] = theme.Current.SubTabActive.Render(subTabs[i])
				} else {
					subTabs[i] = theme.Current.SubTabInactive.Render(subTabs[i])
				}
			}
			headerBar := strings.Join(subTabs, " | ")
			badge := ""
			if selectedNode.Type == NodeProject {
				badge = "  " + theme.Current.TreeProject.Render("Project: "+selectedNode.Name)
			} else {
				badge = "  " + theme.Current.TreeService.Render("Service: "+selectedNode.Name)
			}
			headerBar += badge

			var rightBody string
			switch m.composeActivePane {
			case ComposeSubPaneConfig:
				rightBody = m.composePane.Render()
			case ComposeSubPaneLogs:
				rightBody = m.logsPane.Render()
			}
			rightContent = headerBar + "\n" + rightBody
		}
	} else {
		// Container selected -> Render sub-tab header and active subpane
		subTabs := []string{"[Stats]", "[Logs]", "[Top]", "[Inspect]"}
		for i := range subTabs {
			if SubPane(i) == m.activePane {
				subTabs[i] = theme.Current.SubTabActive.Render(subTabs[i])
			} else {
				subTabs[i] = theme.Current.SubTabInactive.Render(subTabs[i])
			}
		}
		headerBar := strings.Join(subTabs, " | ")

		var selectedContainer *docker.ContainerInfo
		if selectedNode != nil {
			selectedContainer = selectedNode.Container
		}

		var rightBody string
		switch m.activePane {
		case SubPaneStats:
			var actionLoading string
			var actionType string
			if pruneAct, ok := m.runningActions[ActionTargetAll]; ok {
				actionLoading = m.spinner.View() + " " + theme.Current.ActionLoading.Render(pruneAct.Description)
				actionType = pruneAct.Action.String()
			} else if selectedContainer != nil {
				if act, ok := m.runningActions[selectedContainer.ID]; ok {
					actionLoading = m.spinner.View() + " " + theme.Current.ActionLoading.Render(act.Description)
					actionType = act.Action.String()
				}
			}
			rightBody = m.statsPane.Render(selectedContainer, rightWidth-2, m.height-4, actionLoading, actionType, m.spinner.View())
		case SubPaneLogs:
			rightBody = m.logsPane.Render()
		case SubPaneTop:
			rightBody = m.topPane.Render()
		case SubPaneInspect:
			rightBody = m.inspectPane.Render()
		}

		rightContent = headerBar + "\n" + rightBody
	}

	innerLeftWidth = max(leftWidth-2, 1)
	innerRightWidth := max(rightWidth-2, 1)
	maxLines := max(m.height-2, 1)

	// Clamp left pane: truncate lines that exceed the pane width and drop
	// lines beyond maxLines.
	leftStr := layout.ClampLines(leftContent.String(), innerLeftWidth, maxLines)

	// Same for right pane.
	rightContent = layout.ClampLines(rightContent, innerRightWidth, maxLines)

	baseView := lipgloss.JoinHorizontal(lipgloss.Top,
		leftStyle.Render(leftStr),
		rightStyle.Render(rightContent),
	)

	if m.confirmModal.Active {
		return m.confirmModal.Overlay(baseView, m.width, m.height)
	}

	return baseView
}

// FooterBindings returns the active keybindings relevant to the current container view state.
func (m Model) FooterBindings() []key.Binding {
	if m.confirmModal.Active {
		return m.confirmModal.FooterBindings()
	}

	if m.searchBar.IsActive() {
		return []key.Binding{
			keys.CommonHelpKeys.HelpFilterActive,
		}
	}

	selectedNode := m.SelectedNode()
	var bindings []key.Binding

	bindings = append(bindings, keys.AppKeys.FocusToggle)

	// If a project or service is selected (docker-compose config / logs view)
	if selectedNode != nil && (selectedNode.Type == NodeProject || selectedNode.Type == NodeService) {
		if m.focusZone == FocusLeftTree {

			bindings = append(bindings,
				keys.CommonHelpKeys.HelpNavigate,
				keys.CommonHelpKeys.HelpFilter,
				keys.ContainerKeys.ToggleExpand,
			)
		}
		if selectedNode.Name != StandaloneProjectName {
			bindings = append(bindings, keys.CommonHelpKeys.HelpSubpane)
		}
		if m.focusZone == FocusRightPane {
			if m.composeActivePane == ComposeSubPaneLogs {
				bindings = append(bindings, keys.ContainerKeys.FollowLogs, keys.ContainerKeys.TopLogs)
			}
			bindings = append(bindings, keys.CommonHelpKeys.HelpScrollBoth, keys.ContainerKeys.ToggleWrap)
		}
		bindings = append(bindings, keys.AppKeys.Quit)
		return bindings
	}

	// Container is selected
	if m.focusZone == FocusLeftTree {
		bindings = append(bindings,
			keys.CommonHelpKeys.HelpMove,
			keys.ContainerKeys.ToggleExpand,
		)
	}

	// Subpane cycling
	bindings = append(bindings,
		keys.CommonHelpKeys.HelpSubpane,
	)

	// Specific to active subpane
	switch m.activePane {
	case SubPaneStats:
		selected := m.SelectedContainer()
		if selected != nil {
			switch selected.State {
			case docker.StateRunning:
				bindings = append(bindings,
					keys.ContainerKeys.Stop,
					keys.ContainerKeys.Pause,
					keys.ContainerKeys.Restart,
					keys.ContainerKeys.Remove,
					keys.ContainerKeys.Prune,
					keys.ContainerKeys.Exec,
				)
			case docker.StatePaused:
				bindings = append(bindings,
					keys.ContainerKeys.Unpause,
					keys.ContainerKeys.Stop,
					keys.ContainerKeys.Restart,
					keys.ContainerKeys.Remove,
					keys.ContainerKeys.Prune,
				)
			default: // stopped, exited, created
				bindings = append(bindings,
					keys.ContainerKeys.Start,
					keys.ContainerKeys.Restart,
					keys.ContainerKeys.Remove,
					keys.ContainerKeys.Prune,
				)
			}
			bindings = append(bindings, keys.ContainerKeys.ToggleBraille)
		}

	case SubPaneLogs:
		bindings = append(bindings,
			keys.ContainerKeys.FollowLogs,
			keys.ContainerKeys.ToggleWrap,
		)
		if m.focusZone == FocusRightPane {
			bindings = append(bindings, keys.CommonHelpKeys.HelpScrollUp)
		}

	case SubPaneTop:
		bindings = append(bindings,
			keys.ContainerKeys.ToggleWrap,
		)
		if m.focusZone == FocusRightPane {
			bindings = append(bindings, keys.CommonHelpKeys.HelpScroll)
		}

	case SubPaneInspect:
		bindings = append(bindings,
			keys.ContainerKeys.ToggleRaw,
			keys.ContainerKeys.ToggleWrap,
		)
		if m.focusZone == FocusRightPane {
			bindings = append(bindings, keys.CommonHelpKeys.HelpScroll)
		}
	}

	bindings = append(bindings, keys.AppKeys.Quit)
	return bindings
}
