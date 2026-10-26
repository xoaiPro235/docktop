package image

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

	leftWidth := (m.width * 52) / 100
	rightWidth := m.width - leftWidth

	leftBorderColor := theme.Current.Palette.BorderNormal
	rightBorderColor := theme.Current.Palette.BorderNormal

	if m.focusZone == FocusImagesList {
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

	// --- 1. Render Left Images Table ---
	innerLeftWidth := max(leftWidth-2, 10)

	idColWidth := 10
	sizeColWidth := 10
	statusColWidth := 10
	cursorColWidth := 2

	tagColWidth := max(innerLeftWidth-cursorColWidth-idColWidth-sizeColWidth-statusColWidth-3, 14)

	var leftSB strings.Builder

	if m.searchBar.IsActive() || m.searchBar.HasFilter() {
		leftSB.WriteString(m.searchBar.View(innerLeftWidth))
		leftSB.WriteString("\n\n")
	}

	// Table Header
	headerTag := theme.Current.TreeHeader.Render(fmt.Sprintf("%-*s", tagColWidth, "REPOSITORY:TAG"))
	headerID := theme.Current.TreeHeader.Render(fmt.Sprintf("%-*s", idColWidth, "ID"))
	headerSize := theme.Current.TreeHeader.Render(fmt.Sprintf("%-*s", sizeColWidth, "SIZE"))
	headerStatus := theme.Current.TreeHeader.Render(fmt.Sprintf("%-*s", statusColWidth, "STATUS"))

	fmt.Fprintf(&leftSB, "  %s %s %s %s\n\n", headerTag, headerID, headerSize, headerStatus)

	if len(m.filteredIndices) == 0 {
		if m.searchBar.HasFilter() {
			leftSB.WriteString("  ")
			leftSB.WriteString(theme.Current.MutedText.Render(fmt.Sprintf("No images matching %q", m.searchBar.Value())))
			leftSB.WriteString("\n")
		} else {
			leftSB.WriteString(theme.Current.MutedText.Padding(1).Render("No images found."))
			leftSB.WriteString("\n")
		}
	} else {
		startIdx, endIdx, _ := layout.ListWindow(
			m.height,
			m.cursorIndex,
			len(m.filteredIndices),
			m.searchBar.IsActive() || m.searchBar.HasFilter(),
		)

		for i := startIdx; i < endIdx; i++ {
			actualIdx := m.filteredIndices[i]
			img := m.images[actualIdx]

			var cursorMark string
			if i == m.cursorIndex {
				cursorMark = "> "
			} else {
				cursorMark = "  "
			}

			// Styled Tag
			var tagDisplay string
			if img.Dangling {
				tagDisplay = theme.Current.MutedText.Render(ansi.Truncate(img.Tag, tagColWidth, "…"))
			} else {
				parts := strings.SplitN(img.Tag, ":", 2)
				if len(parts) == 2 {
					tagDisplay = theme.Current.Title.Render(parts[0]) + theme.Current.MutedText.Render(":"+parts[1])
				} else {
					tagDisplay = theme.Current.Title.Render(img.Tag)
				}
				tagDisplay = ansi.Truncate(tagDisplay, tagColWidth, "…")
			}
			// Pad to tagColWidth
			tagWidth := ansi.StringWidth(tagDisplay)
			if tagWidth < tagColWidth {
				tagDisplay += strings.Repeat(" ", tagColWidth-tagWidth)
			}

			// Styled ID
			idDisplay := theme.Current.MutedText.Render(fmt.Sprintf("%-*s", idColWidth, docker.CleanImageID(img.ID)))

			// Styled Size
			sizeDisplay := theme.Current.Value.Render(fmt.Sprintf("%-*s", sizeColWidth, docker.FormatImageSize(img.Size)))

			// Styled Status
			var statusDisplay string
			if img.InUse {
				statusDisplay = theme.Current.StatusRunning.Render("● IN-USE")
			} else if img.Dangling {
				statusDisplay = theme.Current.StatusPaused.Render("⚠ DANGL.")
			} else {
				statusDisplay = theme.Current.MutedText.Render("○ UNUSED")
			}
			statusDisplay = fmt.Sprintf("%-*s", statusColWidth, statusDisplay)

			// Combine row
			rowContent := fmt.Sprintf("%s %s %s %s", tagDisplay, idDisplay, sizeDisplay, statusDisplay)

			var line string
			if m.action.IsRunning() && img.ID == m.action.TargetID {
				line = m.spinner.View() + " " + theme.Current.TreeSelected.Render(rowContent)
			} else if i == m.cursorIndex && m.focusZone == FocusImagesList {
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
		leftSB.WriteString(theme.Current.ErrorText.Render(m.action.StatusMsg))
		leftSB.WriteString("\n")
	}

	// --- 2. Render Right Details & Layers ---
	innerRightWidth := max(rightWidth-2, 1)

	rightContent := m.layersViewport.View()

	// Clamp line outputs
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

	if m.searchBar.IsActive() {
		return []key.Binding{
			keys.CommonHelpKeys.HelpFilterActive,
		}
	}

	if m.focusZone == FocusLayersPane {
		return []key.Binding{
			keys.AppKeys.FocusToggle,
			keys.CommonHelpKeys.HelpScroll,
			keys.CommonHelpKeys.HelpScrollX,
			keys.ImageKeys.ToggleWrap,
			keys.ImageKeys.Remove,
			keys.ImageKeys.Prune,
			keys.AppKeys.Quit,
		}
	}

	return []key.Binding{
		keys.AppKeys.FocusToggle,
		keys.CommonHelpKeys.HelpMove,
		keys.CommonHelpKeys.HelpFilter,
		keys.ImageKeys.Remove,
		keys.ImageKeys.Prune,
		keys.AppKeys.Quit,
	}
}
