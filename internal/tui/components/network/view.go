package network

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/xoaiPro235/docktop/internal/tui/keys"
	"github.com/xoaiPro235/docktop/internal/tui/layout"
	"github.com/xoaiPro235/docktop/internal/tui/theme"
)

func (m Model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	leftWidth := (m.width * 52) / 100
	rightWidth := m.width - leftWidth

	leftBorderColor := theme.Current.Palette.BorderNormal
	rightBorderColor := theme.Current.Palette.BorderNormal

	if m.focusZone == FocusNetworksList {
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

	// 1. Render Left Pane (Networks Table)
	innerLeftWidth := max(leftWidth-2, 10)

	cursorColWidth := 2
	driverColWidth := 8
	scopeColWidth := 7
	connColWidth := 14

	nameColWidth := max(innerLeftWidth-cursorColWidth-driverColWidth-scopeColWidth-connColWidth-6, 14)

	var leftSB strings.Builder

	headerName := theme.Current.TreeHeader.Render(fmt.Sprintf("%-*s", nameColWidth, "NETWORK NAME"))
	headerDriver := theme.Current.TreeHeader.Render(fmt.Sprintf("%-*s", driverColWidth, "DRIVER"))
	headerScope := theme.Current.TreeHeader.Render(fmt.Sprintf("%-*s", scopeColWidth, "SCOPE"))
	headerConn := theme.Current.TreeHeader.Render(fmt.Sprintf("%-*s", connColWidth, "CONTAINERS"))

	fmt.Fprintf(&leftSB, "  %s  %s  %s  %s\n\n", headerName, headerDriver, headerScope, headerConn)

	if len(m.networks) == 0 {
		leftSB.WriteString(theme.Current.MutedText.Padding(1).Render("No networks found."))
		leftSB.WriteString("\n")
	} else {
		maxVisibleRows := max(m.height-4, 1)
		startIndex := 0
		if m.cursorIndex >= maxVisibleRows {
			startIndex = m.cursorIndex - maxVisibleRows + 1
		}
		endIndex := min(len(m.networks), startIndex+maxVisibleRows)

		for i := startIndex; i < endIndex; i++ {
			netItem := m.networks[i]

			var cursorMark string
			if i == m.cursorIndex {
				cursorMark = "> "
			} else {
				cursorMark = "  "
			}

			// Format columns: pad plain string first, then style
			nameTrunc := ansi.Truncate(netItem.Name, nameColWidth, "…")
			namePad := fmt.Sprintf("%-*s", nameColWidth, nameTrunc)
			var nameDisplay string
			if netItem.ConnectedCount > 0 {
				nameDisplay = theme.Current.Title.Render(namePad)
			} else {
				nameDisplay = theme.Current.MutedText.Render(namePad)
			}

			driverDisplay := theme.Current.Value.Render(fmt.Sprintf("%-*s", driverColWidth, netItem.Driver))
			scopeDisplay := theme.Current.Key.Render(fmt.Sprintf("%-*s", scopeColWidth, netItem.Scope))

			var connRaw string
			var connStyle lipgloss.Style
			if netItem.ConnectedCount > 0 {
				connRaw = fmt.Sprintf("%d attached", netItem.ConnectedCount)
				connStyle = theme.Current.StatusRunning
			} else {
				connRaw = "0 attached"
				connStyle = theme.Current.MutedText
			}
			connWidth := ansi.StringWidth(connRaw)
			if connWidth < connColWidth {
				connRaw += strings.Repeat(" ", connColWidth-connWidth)
			}
			connDisplay := connStyle.Render(connRaw)

			rowContent := fmt.Sprintf("%s  %s  %s  %s", nameDisplay, driverDisplay, scopeDisplay, connDisplay)

			var line string
			if m.action.IsRunning() && netItem.ID == m.action.TargetID {
				line = m.spinner.View() + " " + theme.Current.TreeSelected.Render(rowContent)
			} else if i == m.cursorIndex && m.focusZone == FocusNetworksList {
				line = theme.Current.TreeSelected.Render(cursorMark + rowContent)
			} else if i == m.cursorIndex {
				line = theme.Current.AccentText.Render(cursorMark) + rowContent
			} else {
				line = cursorMark + rowContent
			}

			leftSB.WriteString(line)
			leftSB.WriteString("\n")
		}
	}

	if m.action.StatusMsg != "" {
		leftSB.WriteString("\n")
		leftSB.WriteString(m.action.StatusMsg)
		leftSB.WriteString("\n")
	}

	// 2. Render Right Details
	innerRightWidth := max(rightWidth-2, 1)

	rightContent := m.viewport.View()

	// Clamp line outputs so bottom borders are perfectly matched
	maxLines := max(m.height-2, 1)

	leftStr := layout.ClampLines(leftSB.String(), innerLeftWidth, maxLines)
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

func (m Model) FooterBindings() []key.Binding {
	if m.confirmModal.Active {
		return m.confirmModal.FooterBindings()
	}

	if m.focusZone == FocusDetailsPane {
		return []key.Binding{
			keys.AppKeys.FocusToggle,
			keys.CommonHelpKeys.HelpScroll,
			keys.CommonHelpKeys.HelpScrollX,
			keys.NetworkKeys.ToggleWrap,
			keys.NetworkKeys.Delete,
			keys.NetworkKeys.Prune,
			keys.AppKeys.Quit,
		}
	}

	return []key.Binding{
		keys.AppKeys.FocusToggle,
		keys.CommonHelpKeys.HelpMove,
		keys.NetworkKeys.Delete,
		keys.NetworkKeys.Prune,
		keys.AppKeys.Quit,
	}
}
