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
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	leftWidth := m.leftWidth()
	rightWidth := m.rightWidth()

	leftStyle := theme.Current.PaneBorder(m.focusZone == FocusLeftTree).
		Width(leftWidth).
		Height(m.height).
		MaxHeight(m.height)

	rightStyle := theme.Current.PaneBorder(m.focusZone != FocusLeftTree).
		Width(rightWidth).
		Height(m.height).
		MaxHeight(m.height)

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
			m.scrollOffset,
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
				if m.runningIDs[item.Node.Key] {
					styledTitle = theme.Current.ActionLoading.Render(m.spinner.View() + " " + title)
				} else {
					styledTitle = theme.Current.TreeProject.Render(title)
				}
			case NodeService:
				if m.runningIDs[item.Node.Key] {
					styledTitle = theme.Current.ActionLoading.Render(m.spinner.View() + " " + title)
				} else {
					styledTitle = theme.Current.TreeService.Render(title)
				}
			case NodeContainer:
				if item.Node.Container != nil {
					if m.runningIDs[item.Node.Container.ID] {
						styledTitle = theme.Current.ActionLoading.Render(m.spinner.View() + " " + title)
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
			isRunning := selectedContainer != nil && m.runningIDs[selectedContainer.ID]
			rightBody = m.statsPane.Render(selectedContainer, rightWidth-2, m.height-4, isRunning, m.spinner.View())
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

	if m.isLeftZoomed() {
		return leftStyle.Render(leftStr)
	}
	if m.isRightZoomed() {
		return rightStyle.Render(rightContent)
	}

	baseView := lipgloss.JoinHorizontal(lipgloss.Top,
		leftStyle.Render(leftStr),
		rightStyle.Render(rightContent),
	)

	return baseView
}

// FooterBindings returns the active keybindings relevant to the current container view state.
func (m Model) FooterBindings() []key.Binding {
	if m.searchBar.IsActive() {
		return []key.Binding{
			keys.CommonHelpKeys.HelpFilterActive,
		}
	}

	selectedNode := m.SelectedNode()
	var bindings []key.Binding

	// If a compose project or service is selected
	if selectedNode != nil && (selectedNode.Type == NodeProject || selectedNode.Type == NodeService) {
		if selectedNode.IsCompose() {
			if selectedNode.HasRunningContainers() {
				bindings = append(bindings, keys.ComposeKeys.Stop, keys.ComposeKeys.Restart)
			} else {
				bindings = append(bindings, keys.ComposeKeys.Start, keys.ComposeKeys.Restart)
			}
		}
		if selectedNode.Name != StandaloneProjectName {
			bindings = append(bindings, keys.CommonHelpKeys.HelpSubpane)
		}
		bindings = append(bindings, keys.AppKeys.Menu, keys.AppKeys.Quit)
		return bindings
	}

	// Container is selected
	selected := m.SelectedContainer()
	if selected != nil {
		switch selected.State {
		case docker.StateRunning:
			bindings = append(bindings,
				keys.ContainerKeys.Stop,
				keys.ContainerKeys.Restart,
				keys.ContainerKeys.Remove,
			)
		case docker.StatePaused:
			bindings = append(bindings,
				keys.ContainerKeys.Unpause,
				keys.ContainerKeys.Stop,
				keys.ContainerKeys.Remove,
			)
		default: // stopped, exited, created
			bindings = append(bindings,
				keys.ContainerKeys.Start,
				keys.ContainerKeys.Remove,
			)
		}
	}

	// Subpane cycling & Navigation hint
	bindings = append(bindings, keys.CommonHelpKeys.HelpSubpane)

	if m.activePane == SubPaneLogs {
		bindings = append(bindings, keys.ContainerKeys.FollowLogs)
	}

	bindings = append(bindings, keys.AppKeys.Menu, keys.AppKeys.Quit)
	return bindings
}
