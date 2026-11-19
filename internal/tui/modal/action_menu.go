package modal

import (
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/xoaiPro235/docktop/internal/tui/keys"
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
	Title         string
	Actions       []ActionItem
	CustomCmds    []ActionItem
	AppActions    []ActionItem
	SelectedIndex int
}

// NewActionMenuModal returns a new initialized ActionMenuModal.
func NewActionMenuModal() ActionMenuModal {
	return ActionMenuModal{
		Active:        false,
		SelectedIndex: 0,
	}
}

// Show activates the action menu with contextual actions, custom commands, and app actions.
func (m *ActionMenuModal) Show(title string, actions, customCmds, appActions []ActionItem) {
	m.Active = true
	m.Title = title
	m.Actions = actions
	m.CustomCmds = customCmds
	m.AppActions = appActions
	m.SelectedIndex = 0
}

// Hide closes the action menu and clears its items.
func (m *ActionMenuModal) Hide() {
	m.Active = false
	m.Actions = nil
	m.CustomCmds = nil
	m.AppActions = nil
	m.SelectedIndex = 0
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
			return m, nil

		case key.Matches(msg, km.Down):
			if m.SelectedIndex < len(items)-1 {
				m.SelectedIndex++
			} else {
				m.SelectedIndex = 0
			}
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
			// Check if pressed key matches any item's shortcut key directly
			pressed := msg.String()
			for _, item := range items {
				if item.Key != "" && strings.EqualFold(item.Key, pressed) {
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

	// 2. Render hardcoded groups: Actions, Custom Commands, Application
	groups := []struct {
		header string
		items  []ActionItem
	}{
		{header: "ACTIONS", items: m.Actions},
		{header: "CUSTOM COMMANDS", items: m.CustomCmds},
		{header: "APPLICATION", items: m.AppActions},
	}

	flatIdx := 0
	hasPrevGroup := false
	for _, g := range groups {
		if len(g.items) == 0 {
			continue
		}
		if hasPrevGroup {
			sb.WriteString("\n")
		}
		catHeader := theme.Current.MutedText.Bold(true).Render("─ " + g.header + " ─")
		sb.WriteString(catHeader)
		sb.WriteString("\n")
		hasPrevGroup = true

		for _, item := range g.items {
			isSelected := flatIdx == m.SelectedIndex

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

			sb.WriteString(rowContent)
			sb.WriteString("\n")
			flatIdx++
		}
	}

	sb.WriteString("\n")

	// 3. Command Preview Bar (If menu contains any custom commands)
	items := m.Items()
	hasAnyCommand := false
	for _, it := range items {
		if it.Command != "" {
			hasAnyCommand = true
			break
		}
	}

	if hasAnyCommand {
		sb.WriteString(theme.Current.MutedText.Render(strings.Repeat("─", innerWidth)))
		sb.WriteString("\n")

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

	// 4. Footer Hint
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
	modalStr := m.View()

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
