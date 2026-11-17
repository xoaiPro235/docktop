package volume

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

	leftWidth := m.leftWidth()
	rightWidth := m.rightWidth()

	leftBorderColor := theme.Current.Palette.BorderNormal
	rightBorderColor := theme.Current.Palette.BorderNormal

	if m.focusZone == FocusVolumesList {
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

	// 1. Render Left Pane (Volumes Table)
	innerLeftWidth := max(leftWidth-2, 10)

	cursorColWidth := 2
	driverColWidth := 8
	statusColWidth := 12
	refColWidth := 5

	extraZoomWidth := 0
	zoomedBadge := ""
	if m.isLeftZoomed() {
		extraZoomWidth = 11
		zoomedBadge = "  " + theme.Current.AccentText.Render("[ZOOMED]")
	}

	availableNameWidth := max(innerLeftWidth-cursorColWidth-driverColWidth-statusColWidth-refColWidth-6-extraZoomWidth, 14)
	nameColWidth := availableNameWidth
	if m.isLeftZoomed() {
		nameColWidth = min(availableNameWidth, 50)
	}

	var leftSB strings.Builder

	if m.searchBar.IsActive() || m.searchBar.HasFilter() {
		leftSB.WriteString(m.searchBar.View(innerLeftWidth))
		leftSB.WriteString("\n\n")
	}

	headerName := theme.Current.TreeHeader.Render(fmt.Sprintf("%-*s", nameColWidth, "VOLUME NAME"))
	headerDriver := theme.Current.TreeHeader.Render(fmt.Sprintf("%-*s", driverColWidth, "DRIVER"))
	headerStatus := theme.Current.TreeHeader.Render(fmt.Sprintf("%-*s", statusColWidth, "STATUS"))
	headerRef := theme.Current.TreeHeader.Render(fmt.Sprintf("%-*s", refColWidth, "REF"))

	fmt.Fprintf(&leftSB, "  %s  %s  %s  %s%s\n\n", headerName, headerDriver, headerStatus, headerRef, zoomedBadge)

	if len(m.filteredIndices) == 0 {
		if m.searchBar.HasFilter() {
			leftSB.WriteString("  ")
			leftSB.WriteString(theme.Current.MutedText.Render(fmt.Sprintf("No volumes matching %q", m.searchBar.Value())))
			leftSB.WriteString("\n")
		} else {
			leftSB.WriteString(theme.Current.MutedText.Padding(1).Render("No volumes found."))
			leftSB.WriteString("\n")
		}
	} else {
		startIndex, endIndex, _ := layout.ListWindow(
			m.height,
			m.cursorIndex,
			m.scrollOffset,
			len(m.filteredIndices),
			m.searchBar.IsActive() || m.searchBar.HasFilter(),
		)

		for i := startIndex; i < endIndex; i++ {
			actualIdx := m.filteredIndices[i]
			vol := m.volumes[actualIdx]

			var cursorMark string
			if i == m.cursorIndex {
				cursorMark = "> "
			} else {
				cursorMark = "  "
			}

			// Format columns: pad plain string first, then style
			nameTrunc := ansi.Truncate(vol.Name, nameColWidth, "…")
			namePad := fmt.Sprintf("%-*s", nameColWidth, nameTrunc)
			var nameDisplay string
			if vol.InUse {
				nameDisplay = theme.Current.Title.Render(namePad)
			} else {
				nameDisplay = theme.Current.MutedText.Render(namePad)
			}

			driverDisplay := theme.Current.Value.Render(fmt.Sprintf("%-*s", driverColWidth, vol.Driver))

			var statusRaw string
			var statusStyle lipgloss.Style
			if vol.InUse {
				statusRaw = "● IN-USE"
				statusStyle = theme.Current.StatusRunning
			} else {
				statusRaw = "⚠ DANGLING"
				statusStyle = theme.Current.StatusStopped
			}
			stWidth := ansi.StringWidth(statusRaw)
			if stWidth < statusColWidth {
				statusRaw += strings.Repeat(" ", statusColWidth-stWidth)
			}
			statusDisplay := statusStyle.Render(statusRaw)

			refRaw := fmt.Sprintf("(%d)", vol.RefCnt)
			refPad := fmt.Sprintf("%-*s", refColWidth, refRaw)
			refDisplay := theme.Current.Key.Render(refPad)

			rowContent := fmt.Sprintf("%s  %s  %s  %s", nameDisplay, driverDisplay, statusDisplay, refDisplay)

			var line string
			if m.runningIDs[vol.Name] {
				line = theme.Current.ActionLoading.Render(m.spinner.View() + " " + rowContent)
			} else if i == m.cursorIndex && m.focusZone == FocusVolumesList {
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

	// 2. Render Right Pane (Details Viewport)
	innerRightWidth := max(rightWidth-2, 1)

	rightContent := m.viewport.View()

	// Clamp line outputs so bottom borders are perfectly matched
	maxLines := max(m.height-2, 1)

	leftStr := layout.ClampLines(leftSB.String(), innerLeftWidth, maxLines)
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

func (m Model) FooterBindings() []key.Binding {
	if m.searchBar.IsActive() {
		return []key.Binding{
			keys.CommonHelpKeys.HelpFilterActive,
		}
	}

	var bindings []key.Binding
	bindings = append(bindings, keys.AppKeys.FocusToggle)
	if m.isZoomed {
		bindings = append(bindings, keys.CommonHelpKeys.HelpUnzoom)
	} else {
		bindings = append(bindings, keys.CommonHelpKeys.HelpZoom)
	}

	if m.focusZone == FocusDetailsPane {
		bindings = append(bindings,
			keys.CommonHelpKeys.HelpScroll,
			keys.CommonHelpKeys.HelpScrollX,
			keys.VolumeKeys.ToggleWrap,
			keys.VolumeKeys.Remove,
			keys.VolumeKeys.Prune,
			keys.AppKeys.Quit,
		)
		return bindings
	}

	bindings = append(bindings,
		keys.CommonHelpKeys.HelpMove,
		keys.CommonHelpKeys.HelpFilter,
		keys.VolumeKeys.Remove,
		keys.VolumeKeys.Prune,
		keys.AppKeys.Quit,
	)
	return bindings
}
