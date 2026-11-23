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

// Style represents the visual style/severity of the confirmation dialog.
type Style int

const (
	StyleDefault Style = iota
	StyleDanger
	StyleWarning
	StyleSuccess
)

// ShowConfirmMsg is sent to AppModel to display a confirmation dialog.
type ShowConfirmMsg struct {
	Question  string
	Command   string
	Style     Style
	OnConfirm tea.Cmd
}

// ConfirmModal manages a floating confirmation dialog with command preview.
type ConfirmModal struct {
	Active      bool
	Question    string
	Command     string
	Style       Style
	SelectedYes bool
	OnConfirm   tea.Cmd
}

func NewConfirmModal() ConfirmModal {
	return ConfirmModal{
		SelectedYes: true,
	}
}

// Show activates the confirmation dialog.
func (m *ConfirmModal) Show(question, command string, style Style, onConfirm tea.Cmd) {
	m.Active = true
	m.Question = question
	m.Command = command
	m.Style = style
	m.SelectedYes = true
	m.OnConfirm = onConfirm
}

// Hide closes the modal.
func (m *ConfirmModal) Hide() {
	m.Active = false
	m.OnConfirm = nil
}

func (m ConfirmModal) Init() tea.Cmd {
	return nil
}

func (m ConfirmModal) Update(msg tea.Msg) (ConfirmModal, tea.Cmd) {
	if !m.Active {
		return m, nil
	}

	km := keys.ConfirmModalKeys

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, km.Yes):
			cmd := m.OnConfirm
			m.Hide()
			return m, cmd

		case key.Matches(msg, km.No):
			m.Hide()
			return m, nil

		case key.Matches(msg, km.Switch):
			m.SelectedYes = !m.SelectedYes
			return m, nil

		case key.Matches(msg, km.Enter):
			if m.SelectedYes {
				cmd := m.OnConfirm
				m.Hide()
				return m, cmd
			}
			m.Hide()
			return m, nil
		}
	}

	return m, nil
}

func (m ConfirmModal) View() string {
	boxWidth := 52
	innerWidth := boxWidth - 6 // 52 - 2(borders) - 4(paddings) = 46 exact content space

	borderColor := theme.Current.Palette.Primary
	switch m.Style {
	case StyleDanger:
		borderColor = theme.Current.Palette.Danger
	case StyleWarning:
		borderColor = theme.Current.Palette.Warning
	case StyleSuccess:
		borderColor = theme.Current.Palette.Success
	}

	var sb strings.Builder

	// 1. Question (clean and centered)
	sb.WriteString(lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Center).Render(m.Question))
	sb.WriteString("\n\n")

	// 2. Command Preview Box (rendered only if command is provided)
	if m.Command != "" {
		cmdContent := theme.Current.MutedText.Render("$ ") + theme.Current.AccentText.Bold(true).Render(m.Command)
		cmdBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(theme.Current.Palette.BorderNormal).
			Width(innerWidth).
			Padding(0, 1).
			Render(cmdContent)

		sb.WriteString(cmdBox)
		sb.WriteString("\n\n")
	}

	// 3. Buttons (Yes / No)
	var yesBtn, noBtn string
	if m.SelectedYes {
		yesBtn = theme.Current.BtnPrimary.Render("✔ Yes")
		noBtn = theme.Current.BtnInactive.Render("✖ No")
	} else {
		yesBtn = theme.Current.BtnInactive.Render("✔ Yes")
		noBtn = theme.Current.BtnDanger.Render("✖ No")
	}

	btnRow := lipgloss.JoinHorizontal(lipgloss.Center, yesBtn, "     ", noBtn)
	btnCentered := lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Center).Render(btnRow)
	sb.WriteString(btnCentered)
	sb.WriteString("\n\n")

	// 4. Compact, subtle shortcut hint
	hintText := "y: Yes  •  n/Esc: No  •  Tab: Switch"
	sb.WriteString(theme.Current.HintText.Width(innerWidth).Align(lipgloss.Center).Render(hintText))

	// Outer Dialog Box
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(1, 2).
		Width(boxWidth).
		Render(sb.String())
}

// Overlay draws the modal centered over an existing background screen string.
func (m ConfirmModal) Overlay(bg string, totalWidth, totalHeight int) string {
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
		bgLines = append(bgLines, "")
	}

	var result []string
	for row := 0; row < totalHeight && row < len(bgLines); row++ {
		bgLine := bgLines[row]
		if row < top || row >= top+modalHeight {
			result = append(result, bgLine)
			continue
		}

		modalLine := modalLines[row-top]
		leftPart := ansi.Cut(bgLine, 0, left)
		leftLen := ansi.StringWidth(leftPart)
		if leftLen < left {
			leftPart += strings.Repeat(" ", left-leftLen)
		}

		rightPart := ansi.Cut(bgLine, left+modalWidth, ansi.StringWidth(bgLine))
		result = append(result, leftPart+modalLine+rightPart)
	}

	return strings.Join(result, "\n")
}

func (m ConfirmModal) FooterBindings() []key.Binding {
	km := keys.ConfirmModalKeys
	return []key.Binding{
		km.Switch,
		key.NewBinding(key.WithHelp("y / Enter", "Yes")),
		key.NewBinding(key.WithHelp("n / Esc", "No")),
	}
}
