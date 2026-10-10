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

type Action int

const (
	ActionNone Action = iota
	ActionStart
	ActionStop
	ActionRestart
	ActionPause
	ActionUnpause
	ActionRemove
	ActionRemoveForce
	ActionPrune
	ActionDelete
)

type Style int

const (
	StyleDefault Style = iota
	StyleDanger
	StyleWarning
	StyleSuccess
)

func (a Action) Style() Style {
	switch a {
	case ActionStop, ActionRemove, ActionRemoveForce, ActionDelete:
		return StyleDanger
	case ActionRestart, ActionPause, ActionPrune:
		return StyleWarning
	case ActionStart, ActionUnpause:
		return StyleSuccess
	default:
		return StyleDefault
	}
}

func (a Action) String() string {
	switch a {
	case ActionStart:
		return "start"
	case ActionStop:
		return "stop"
	case ActionRestart:
		return "restart"
	case ActionPause:
		return "pause"
	case ActionUnpause:
		return "unpause"
	case ActionRemove:
		return "remove"
	case ActionRemoveForce:
		return "remove_force"
	case ActionPrune:
		return "prune"
	case ActionDelete:
		return "delete"
	default:
		return ""
	}
}

func (a Action) Title() string {
	switch a {
	case ActionStart:
		return "Start"
	case ActionStop:
		return "Stop"
	case ActionRestart:
		return "Restart"
	case ActionPause:
		return "Pause"
	case ActionUnpause:
		return "Unpause"
	case ActionRemove:
		return "Remove"
	case ActionRemoveForce:
		return "Force Remove"
	case ActionPrune:
		return "Prune"
	case ActionDelete:
		return "Delete"
	default:
		return ""
	}
}

// ActionState encapsulates the runtime execution state of an asynchronous action.
type ActionState struct {
	Running    bool
	Action     Action
	TargetID   string
	TargetName string
	Desc       string
	StatusMsg  string
}

// Start marks the action as running with the given parameters.
func (s *ActionState) Start(action Action, targetID, targetName, desc string) {
	s.Running = true
	s.Action = action
	s.TargetID = targetID
	s.TargetName = targetName
	s.Desc = desc
	s.StatusMsg = ""
}

// Finish marks the action as finished with an optional status message.
func (s *ActionState) Finish(statusMsg string) {
	s.Running = false
	s.Action = ActionNone
	s.TargetID = ""
	s.TargetName = ""
	s.Desc = ""
	s.StatusMsg = statusMsg
}

// IsRunning returns whether an action is currently active.
func (s ActionState) IsRunning() bool {
	return s.Running
}

// ConfirmModal manages a floating confirmation dialog with command preview.
type ConfirmModal struct {
	Active      bool
	Confirmed   bool // true if confirmed during the last Update
	Question    string
	Command     string
	Style       Style
	Action      Action
	TargetID    string
	TargetName  string
	SelectedYes bool
}

func NewConfirmModal() ConfirmModal {
	return ConfirmModal{
		SelectedYes: true,
	}
}

// Show activates the confirmation dialog with full parameters.
func (m *ConfirmModal) Show(question, command string, style Style, action Action, targetID, targetName string) {
	m.Active = true
	m.Confirmed = false
	m.Question = question
	m.Command = command
	m.Style = style
	m.Action = action
	m.TargetID = targetID
	m.TargetName = targetName
	m.SelectedYes = true
}

// ShowContainerAction formats and shows a container lifecycle confirmation dialog.
func (m *ConfirmModal) ShowContainerAction(action Action, id, name string) {
	var question, command string
	switch action {
	case ActionStart:
		question = fmt.Sprintf("Are you sure you want to start \"%s\"?", name)
		command = fmt.Sprintf("docker start %s", name)
	case ActionStop:
		question = fmt.Sprintf("Are you sure you want to stop \"%s\"?", name)
		command = fmt.Sprintf("docker stop %s", name)
	case ActionPause:
		question = fmt.Sprintf("Are you sure you want to pause \"%s\"?", name)
		command = fmt.Sprintf("docker pause %s", name)
	case ActionUnpause:
		question = fmt.Sprintf("Are you sure you want to unpause \"%s\"?", name)
		command = fmt.Sprintf("docker unpause %s", name)
	case ActionRestart:
		question = fmt.Sprintf("Are you sure you want to restart \"%s\"?", name)
		command = fmt.Sprintf("docker restart %s", name)
	case ActionRemove:
		question = fmt.Sprintf("Are you sure you want to remove container \"%s\"?", name)
		command = fmt.Sprintf("docker rm %s", name)
	case ActionRemoveForce:
		question = fmt.Sprintf("Container is running! Force remove \"%s\"?", name)
		command = fmt.Sprintf("docker rm -f %s", name)
	case ActionPrune:
		question = "Are you sure you want to remove all stopped containers?"
		command = "docker container prune -f"
		id = "all"
		name = "all stopped containers"
	}
	m.Show(question, command, action.Style(), action, id, name)
}

// ShowImageAction formats and shows an image confirmation dialog.
func (m *ConfirmModal) ShowImageAction(action Action, id, ref string) {
	var question, command string
	switch action {
	case ActionRemove:
		question = fmt.Sprintf("Are you sure you want to remove image \"%s\"?", ref)
		command = fmt.Sprintf("docker rmi %s", id)
	case ActionPrune:
		question = "Are you sure you want to remove all dangling images?"
		command = "docker image prune -f"
		id = "all"
	}
	m.Show(question, command, action.Style(), action, id, ref)
}

// ShowVolumeAction formats and shows a volume confirmation dialog.
func (m *ConfirmModal) ShowVolumeAction(action Action, name string) {
	var question, command string
	switch action {
	case ActionRemove:
		question = fmt.Sprintf("Are you sure you want to remove volume \"%s\"?", name)
		command = fmt.Sprintf("docker volume rm %s", name)
	case ActionPrune:
		question = "Are you sure you want to remove all unused volumes?"
		command = "docker volume prune -f"
		name = "all"
	}
	m.Show(question, command, action.Style(), action, name, name)
}

// ShowNetworkAction formats and shows a network confirmation dialog.
func (m *ConfirmModal) ShowNetworkAction(action Action, id, name string) {
	var question, command string
	switch action {
	case ActionDelete, ActionRemove:
		question = fmt.Sprintf("Are you sure you want to remove network \"%s\"?", name)
		command = fmt.Sprintf("docker network rm %s", id)
	case ActionPrune:
		question = "Are you sure you want to remove all unused networks?"
		command = "docker network prune -f"
		id = "all"
		name = "all"
	}
	m.Show(question, command, action.Style(), action, id, name)
}

// Hide closes the modal.
func (m *ConfirmModal) Hide() {
	m.Active = false
	m.Confirmed = false
}

// Init
func (m ConfirmModal) Init() tea.Cmd {
	return nil
}

// Update
func (m ConfirmModal) Update(msg tea.Msg) (ConfirmModal, tea.Cmd) {
	if !m.Active {
		return m, nil
	}

	m.Confirmed = false
	km := keys.ConfirmModalKeys

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, km.Yes):
			m.Active = false
			m.Confirmed = true
			return m, nil
		case key.Matches(msg, km.No):
			m.Active = false
			m.Confirmed = false
			return m, nil
		case key.Matches(msg, km.Switch):
			m.SelectedYes = !m.SelectedYes
			return m, nil
		case key.Matches(msg, km.Enter):
			m.Active = false
			m.Confirmed = m.SelectedYes
			return m, nil
		}
	}
	return m, nil
}

// View
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

	// 1. Question 
	sb.WriteString(lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Center).Render(m.Question))
	sb.WriteString("\n\n")

	// 2. Command Preview Box
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
		yesBtn = lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Current.Palette.Foreground).
			Background(theme.Current.Palette.Success).
			Padding(0, 3).
			Render("✔ Yes")
		noBtn = lipgloss.NewStyle().
			Foreground(theme.Current.Palette.ForegroundDim).
			Background(theme.Current.Palette.BorderNormal).
			Padding(0, 3).
			Render("✖ No")
	} else {
		yesBtn = lipgloss.NewStyle().
			Foreground(theme.Current.Palette.ForegroundDim).
			Background(theme.Current.Palette.BorderNormal).
			Padding(0, 3).
			Render("✔ Yes")
		noBtn = lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.Current.Palette.Foreground).
			Background(theme.Current.Palette.Danger).
			Padding(0, 3).
			Render("✖ No")
	}

	btnRow := lipgloss.JoinHorizontal(lipgloss.Center, yesBtn, "     ", noBtn)
	btnCentered := lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Center).Render(btnRow)
	sb.WriteString(btnCentered)
	sb.WriteString("\n\n")

	// 4. Compact, subtle shortcut hint
	hintText := "y: Yes  •  n/Esc: No  •  Tab: Switch"
	sb.WriteString(lipgloss.NewStyle().Width(innerWidth).Align(lipgloss.Center).Foreground(theme.Current.Palette.TextDark).Render(hintText))

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

// FooterBindings returns the footer key shortcuts while the modal is visible.
func (m ConfirmModal) FooterBindings() []key.Binding {
	km := keys.ConfirmModalKeys
	return []key.Binding{
		km.Switch,
		key.NewBinding(key.WithHelp("y / Enter", "Yes")),
		key.NewBinding(key.WithHelp("n / Esc", "No")),
	}
}



