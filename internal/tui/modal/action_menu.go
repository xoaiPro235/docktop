package modal

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/xoaiPro235/docktop/internal/tui/keys"
	"github.com/xoaiPro235/docktop/internal/tui/layout"
	"github.com/xoaiPro235/docktop/internal/tui/theme"
)

// ActionItem represents a single executable entry in the action menu.
type ActionItem struct {
	Key     string  // Shortcut trigger key, e.g. "s", "r", "d", "1"
	Title   string  // Display title of the action
	Command string  // Optional shell command preview, e.g. "docker exec -it c12345 bash"
	Action  tea.Cmd // Command to execute when triggered
}

// ActionMenuModal manages the popup menu listing all available actions for the selected entity.
type ActionMenuModal struct {
	Active        bool
	TargetID      string
	Title         string
	Actions       []ActionItem
	CustomCmds    []ActionItem
	AppActions    []ActionItem
	SelectedIndex int
	ScrollOffset  int
	Height        int
}

// NewActionMenuModal returns a new initialized ActionMenuModal.
func NewActionMenuModal() ActionMenuModal {
	return ActionMenuModal{
		Active:        false,
		SelectedIndex: 0,
		ScrollOffset:  0,
	}
}

// SetHeight sets the maximum available height and synchronizes scroll position.
func (m *ActionMenuModal) SetHeight(h int) {
	m.Height = h
	m.syncScroll()
}

// Show activates the action menu with contextual actions, custom commands, and app actions.
func (m *ActionMenuModal) Show(title string, actions, customCmds, appActions []ActionItem, targetID ...string) {
	m.Active = true
	m.Title = title
	m.Actions = actions
	m.CustomCmds = customCmds
	m.AppActions = appActions
	m.SelectedIndex = 0
	m.ScrollOffset = 0
	if len(targetID) > 0 {
		m.TargetID = targetID[0]
	} else {
		m.TargetID = ""
	}
	m.syncScroll()
}

// Hide closes the action menu and clears its items.
func (m *ActionMenuModal) Hide() {
	m.Active = false
	m.TargetID = ""
	m.Actions = nil
	m.CustomCmds = nil
	m.AppActions = nil
	m.SelectedIndex = 0
	m.ScrollOffset = 0
}

// Items returns all action items combined in display order.
func (m ActionMenuModal) Items() []ActionItem {
	total := len(m.Actions) + len(m.CustomCmds) + len(m.AppActions)
	items := make([]ActionItem, 0, total)
	items = append(items, m.Actions...)
	items = append(items, m.CustomCmds...)
	items = append(items, m.AppActions...)
	return items
}

type menuVisualRow struct {
	isHeader bool
	flatIdx  int
	content  string
}

func (m ActionMenuModal) hasAnyCommand() bool {
	for _, it := range m.Items() {
		if it.Command != "" {
			return true
		}
	}
	return false
}

func (m ActionMenuModal) availRows(maxHeight int, hasAnyCommand bool) (int, bool) {
	fixedLines := 4 + 2 + 1 + 1 // border(4) + title(2) + blank(1) + footer(1)
	if hasAnyCommand {
		fixedLines += 3 // separator(1) + preview(1) + blank(1)
	}
	if maxHeight > fixedLines+3 {
		return max(maxHeight-fixedLines, 4), true
	}
	return 0, false
}

func (m ActionMenuModal) buildRows(innerWidth int) ([]menuVisualRow, int) {
	groups := []struct {
		header string
		items  []ActionItem
	}{
		{header: "ACTIONS", items: m.Actions},
		{header: "CUSTOM COMMANDS", items: m.CustomCmds},
		{header: "APPLICATION", items: m.AppActions},
	}

	var allRows []menuVisualRow
	flatIdx := 0
	selectedRowIdx := 0
	hasPrevGroup := false

	for _, g := range groups {
		if len(g.items) == 0 {
			continue
		}
		if hasPrevGroup {
			allRows = append(allRows, menuVisualRow{isHeader: true, flatIdx: -1, content: ""})
		}
		catHeader := theme.Current.MutedText.Bold(true).Render("─ " + g.header + " ─")
		allRows = append(allRows, menuVisualRow{isHeader: true, flatIdx: -1, content: catHeader})
		hasPrevGroup = true

		for _, item := range g.items {
			isSelected := flatIdx == m.SelectedIndex
			if isSelected {
				selectedRowIdx = len(allRows)
			}

			keyBadge := lipgloss.NewStyle().
				Foreground(theme.Current.Palette.Primary).
				Bold(true).
				Render("[" + item.Key + "]")

			titleStyle := lipgloss.NewStyle().Foreground(theme.Current.Palette.Foreground)
			if isSelected {
				titleStyle = titleStyle.Bold(true).Foreground(theme.Current.Palette.Accent)
			}
			titleStr := titleStyle.Render(item.Title)

			var rowContent string
			if isSelected {
				cursor := lipgloss.NewStyle().Foreground(theme.Current.Palette.Accent).Bold(true).Render("▸ ")
				rowContent = lipgloss.NewStyle().
					Width(innerWidth).
					Render(cursor + keyBadge + " " + titleStr)
			} else {
				rowContent = lipgloss.NewStyle().
					Width(innerWidth).
					Render("  " + keyBadge + " " + titleStr)
			}

			allRows = append(allRows, menuVisualRow{isHeader: false, flatIdx: flatIdx, content: rowContent})
			flatIdx++
		}
	}
	return allRows, selectedRowIdx
}

func (m *ActionMenuModal) syncScroll() {
	if m.Height <= 0 {
		return
	}
	availRows, constrained := m.availRows(m.Height, m.hasAnyCommand())
	if !constrained || m.SelectedIndex == 0 {
		m.ScrollOffset = 0
		return
	}
	rows, selectedRowIdx := m.buildRows(56)
	if len(rows) <= availRows {
		m.ScrollOffset = 0
		return
	}
	m.ScrollOffset = layout.AdjustOffsetCustom(availRows, selectedRowIdx, m.ScrollOffset, len(rows))
}

// Init initializes the modal.
func (m ActionMenuModal) Init() tea.Cmd {
	return nil
}

// Update handles navigation, quick key triggers, and selection inside the menu.
func (m ActionMenuModal) Update(msg tea.Msg) (ActionMenuModal, tea.Cmd) {
	items := m.Items()
	if !m.Active || len(items) == 0 {
		return m, nil
	}

	km := keys.ActionMenuKeys

	switch msg := msg.(type) {
	case tea.MouseWheelMsg:
		if msg.Button == tea.MouseWheelUp && m.SelectedIndex > 0 {
			m.SelectedIndex--
			m.syncScroll()
		} else if msg.Button == tea.MouseWheelDown && m.SelectedIndex < len(items)-1 {
			m.SelectedIndex++
			m.syncScroll()
		}
		return m, nil

	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, km.Cancel):
			m.Hide()
			return m, nil

		case key.Matches(msg, km.Up):
			if m.SelectedIndex > 0 {
				m.SelectedIndex--
			} else {
				m.SelectedIndex = len(items) - 1
			}
			m.syncScroll()
			return m, nil

		case key.Matches(msg, km.Down):
			if m.SelectedIndex < len(items)-1 {
				m.SelectedIndex++
			} else {
				m.SelectedIndex = 0
			}
			m.syncScroll()
			return m, nil

		case key.Matches(msg, km.Enter):
			if m.SelectedIndex >= 0 && m.SelectedIndex < len(items) {
				cmd := items[m.SelectedIndex].Action
				m.Hide()
				return m, cmd
			}
			m.Hide()
			return m, nil

		default:
			// Check if pressed key matches any item's shortcut key directly (case-sensitive)
			pressed := msg.String()
			for _, item := range items {
				if item.Key != "" && item.Key == pressed {
					cmd := item.Action
					m.Hide()
					return m, cmd
				}
			}
		}
	}

	return m, nil
}

// View renders the styled action menu popup with hardcoded sections: Actions, Custom Commands, Application.
func (m ActionMenuModal) View() string {
	return m.ViewWithHeight(0)
}

// ViewWithHeight renders the action menu fitted to a maximum height, scrolling items to keep SelectedIndex visible.
func (m ActionMenuModal) ViewWithHeight(maxHeight int) string {
	boxWidth := 60
	innerWidth := boxWidth - 4 // 60 - 2(border) - 2(padding) = 56

	var sb strings.Builder

	// 1. Header Title
	titleText := m.Title
	if titleText == "" {
		titleText = "Actions Menu"
	}
	titleFormatted := theme.Current.AccentText.Bold(true).Render(titleText)
	titleCentered := lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Center).Render(titleFormatted)
	sb.WriteString(titleCentered)
	sb.WriteString("\n\n")

	// 2. Build rows for hardcoded groups: Actions, Custom Commands, Application
	allRows, selectedRowIdx := m.buildRows(innerWidth)

	// 3. Determine command preview existence
	hasAnyCommand := m.hasAnyCommand()

	// 4. Calculate available content rows when maxHeight is specified
	visibleRows := allRows
	availRows, constrained := m.availRows(maxHeight, hasAnyCommand)
	if constrained && len(allRows) > availRows {
		startIdx, endIdx := layout.ListWindowCustom(availRows, selectedRowIdx, m.ScrollOffset, len(allRows))
		visibleRows = allRows[startIdx:endIdx]
	}

	for _, r := range visibleRows {
		sb.WriteString(r.content)
		sb.WriteString("\n")
	}

	sb.WriteString("\n")

	// 5. Command Preview Bar (If menu contains any custom commands)
	if hasAnyCommand {
		sb.WriteString(theme.Current.MutedText.Render(strings.Repeat("─", innerWidth)))
		sb.WriteString("\n")

		items := m.Items()
		var selectedCmd string
		if m.SelectedIndex >= 0 && m.SelectedIndex < len(items) {
			selectedCmd = items[m.SelectedIndex].Command
		}

		if selectedCmd != "" {
			prompt := lipgloss.NewStyle().Foreground(theme.Current.Palette.Accent).Bold(true).Render("$ ")
			availWidth := max(innerWidth-ansi.StringWidth(prompt), 10)
			truncatedCmd := ansi.Truncate(selectedCmd, availWidth, "…")
			cmdStyled := lipgloss.NewStyle().Foreground(theme.Current.Palette.Foreground).Render(truncatedCmd)
			sb.WriteString(prompt)
			sb.WriteString(cmdStyled)
		} else {
			sb.WriteString(" ")
		}
		sb.WriteString("\n\n")
	}

	// 6. Footer Hint
	hint := theme.Current.MutedText.Render("j/k: Navigate  •  Enter: Run  •  x/Esc: Close")
	hintCentered := lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Center).Render(hint)
	sb.WriteString(hintCentered)

	// Outer Dialog Box
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Current.Palette.Primary).
		Padding(1, 1).
		Width(boxWidth).
		Render(sb.String())
}

// Overlay draws the modal centered over an existing background screen string.
func (m ActionMenuModal) Overlay(bg string, totalWidth, totalHeight int) string {
	modalStr := m.ViewWithHeight(totalHeight)

	modalLines := strings.Split(modalStr, "\n")
	modalHeight := len(modalLines)
	modalWidth := 0
	for _, l := range modalLines {
		if w := ansi.StringWidth(l); w > modalWidth {
			modalWidth = w
		}
	}

	top := max((totalHeight-modalHeight)/2, 0)
	left := max((totalWidth-modalWidth)/2, 0)

	bgLines := strings.Split(bg, "\n")
	for len(bgLines) < totalHeight {
		bgLines = append(bgLines, strings.Repeat(" ", totalWidth))
	}

	for i, modalLine := range modalLines {
		targetY := top + i
		if targetY >= len(bgLines) {
			break
		}

		bgLine := bgLines[targetY]
		lineWidth := ansi.StringWidth(bgLine)
		if lineWidth < totalWidth {
			bgLine += strings.Repeat(" ", totalWidth-lineWidth)
		}

		leftPart := ansi.Cut(bgLine, 0, left)
		rightPart := ansi.Cut(bgLine, left+ansi.StringWidth(modalLine), totalWidth)
		bgLines[targetY] = leftPart + modalLine + rightPart
	}

	return strings.Join(bgLines, "\n")
}

// FooterBindings returns the keybindings shown in the footer when the modal is active.
func (m ActionMenuModal) FooterBindings() []key.Binding {
	km := keys.ActionMenuKeys
	return []key.Binding{km.Up, km.Down, km.Enter, km.Cancel}
}
