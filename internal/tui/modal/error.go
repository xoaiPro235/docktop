package modal

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/xoaiPro235/docktop/internal/tui/keys"
	"github.com/xoaiPro235/docktop/internal/tui/theme"
)

type ShowErrorMsg struct {
	Title   string
	Message string
}

type ErrorModal struct {
	Active  bool
	Title   string
	Message string
}

func NewErrorModal() ErrorModal {
	return ErrorModal{}
}

// Show activates the error modal with a title and error message.
func (m *ErrorModal) Show(title, message string) {
	m.Active = true
	if title == "" {
		title = "Error"
	}
	m.Title = title
	m.Message = message
}

// Hide closes the error modal.
func (m *ErrorModal) Hide() {
	m.Active = false
}

// Init
func (m ErrorModal) Init() tea.Cmd {
	return nil
}

// Update
func (m ErrorModal) Update(msg tea.Msg) (ErrorModal, tea.Cmd) {
	if !m.Active {
		return m, nil
	}

	km := keys.ErrorModalKeys

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if key.Matches(msg, km.Dismiss) {
			m.Active = false
			return m, nil
		}
	case tea.MouseClickMsg:
		m.Active = false
		return m, nil
	}

	return m, nil
}

// View
func (m ErrorModal) View() string {
	boxWidth := 54
	innerWidth := boxWidth - 6 // 54 - 2(borders) - 4(paddings) = 48 exact content space

	var sb strings.Builder

	// 1. Error Title Banner
	titleText := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current.Palette.Danger).
		Width(innerWidth).
		Align(lipgloss.Center).
		Render(fmt.Sprintf("✖  %s", m.Title))
	sb.WriteString(titleText)
	sb.WriteString("\n\n")

	// 2. Error Message Body
	msgContent := lipgloss.NewStyle().
		Width(innerWidth).
		Foreground(theme.Current.Palette.Foreground).
		Align(lipgloss.Center).
		Render(m.Message)
	sb.WriteString(msgContent)
	sb.WriteString("\n\n")

	// 3. Single Button: "✔ Yes"
	yesBtn := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Current.Palette.Foreground).
		Background(theme.Current.Palette.Danger).
		Padding(0, 4).
		Render("✔ Yes")

	btnCentered := lipgloss.NewStyle().
		Width(innerWidth).
		Align(lipgloss.Center).
		Render(yesBtn)
	sb.WriteString(btnCentered)
	sb.WriteString("\n\n")

	// 4. Shortcut hint
	hintText := lipgloss.NewStyle().
		Width(innerWidth).
		Align(lipgloss.Center).
		Foreground(theme.Current.Palette.TextDark).
		Render("y / Enter / Esc: Dismiss")
	sb.WriteString(hintText)

	// Outer Dialog Box
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(theme.Current.Palette.Danger).
		Padding(1, 2).
		Width(boxWidth).
		Render(sb.String())
}

// Overlay draws the modal centered over an existing background screen string.
func (m ErrorModal) Overlay(bg string, totalWidth, totalHeight int) string {
	modalStr := m.View()

	modalLines := strings.Split(modalStr, "\n")
	modalHeight := len(modalLines)
	modalWidth := 0
	for _, l := range modalLines {
		if w := ansi.StringWidth(l); w > modalWidth {
			modalWidth = w
		}
	}

	top := max((totalHeight - modalHeight) / 2, 0)
	left := max((totalWidth - modalWidth) / 2, 0)

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

// FooterBindings returns the footer key shortcuts while the error modal is visible.
func (m ErrorModal) FooterBindings() []key.Binding {
	return []key.Binding{
		keys.ErrorModalKeys.Dismiss,
	}
}
