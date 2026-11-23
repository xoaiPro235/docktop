package modal

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

type testActionMsg struct {
	id string
}

func TestActionMenuModal_Init(t *testing.T) {
	m := NewActionMenuModal()
	if cmd := m.Init(); cmd != nil {
		t.Errorf("ActionMenuModal.Init() = %v, want nil", cmd)
	}
}

func TestActionMenuModal_Update(t *testing.T) {
	m := NewActionMenuModal()

	// 1. Inactive modal ignores events
	m, cmd := m.Update(tea.KeyPressMsg{Code: 'j'})
	if m.Active || cmd != nil {
		t.Error("Inactive modal should ignore keys")
	}

	actions := []ActionItem{
		{
			Key:    "s",
			Title:  "Stop container",
			Action: func() tea.Msg { return testActionMsg{id: "stop"} },
		},
		{
			Key:    "r",
			Title:  "Restart container",
			Action: func() tea.Msg { return testActionMsg{id: "restart"} },
		},
	}
	customCmds := []ActionItem{
		{
			Key:    "1",
			Title:  "Custom Bash",
			Action: func() tea.Msg { return testActionMsg{id: "custom"} },
		},
	}
	var appActions []ActionItem

	// 2. Navigation Up/Down
	m.Show("Actions: web-app", actions, customCmds, appActions)
	if m.SelectedIndex != 0 {
		t.Errorf("Initial SelectedIndex = %d, want 0", m.SelectedIndex)
	}

	// Down -> 1
	m, _ = m.Update(tea.KeyPressMsg{Code: 'j'})
	if m.SelectedIndex != 1 {
		t.Errorf("SelectedIndex after 'j' = %d, want 1", m.SelectedIndex)
	}

	// Down -> 2
	m, _ = m.Update(tea.KeyPressMsg{Code: 'j'})
	if m.SelectedIndex != 2 {
		t.Errorf("SelectedIndex after 'j' = %d, want 2", m.SelectedIndex)
	}

	// Down wraps -> 0
	m, _ = m.Update(tea.KeyPressMsg{Code: 'j'})
	if m.SelectedIndex != 0 {
		t.Errorf("SelectedIndex after wrapping 'j' = %d, want 0", m.SelectedIndex)
	}

	// Up wraps -> 2
	m, _ = m.Update(tea.KeyPressMsg{Code: 'k'})
	if m.SelectedIndex != 2 {
		t.Errorf("SelectedIndex after wrapping 'k' = %d, want 2", m.SelectedIndex)
	}

	// 3. Trigger via Enter (currently on index 2: custom)
	m, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.Active {
		t.Error("Modal should close after Enter")
	}
	if cmd == nil || cmd() != (testActionMsg{id: "custom"}) {
		t.Errorf("Expected action 'custom', got %v", cmd)
	}

	// 4. Trigger via Direct Key (e.g. 's')
	m.Show("Actions: web-app", actions, customCmds, appActions)
	m, cmd = m.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	if m.Active {
		t.Error("Modal should close after direct key trigger")
	}
	if cmd == nil || cmd() != (testActionMsg{id: "stop"}) {
		t.Errorf("Expected action 'stop', got %v", cmd)
	}

	// 5. Dismiss via Esc
	m.Show("Actions: web-app", actions, customCmds, appActions)
	m, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if m.Active || cmd != nil {
		t.Error("Modal should close without cmd on Esc")
	}

	// 6. Dismiss via 'x'
	m.Show("Actions: web-app", actions, customCmds, appActions)
	m, cmd = m.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	if m.Active || cmd != nil {
		t.Error("Modal should close without cmd on 'x'")
	}
}

func TestActionMenuModal_ViewAndOverlay(t *testing.T) {
	m := NewActionMenuModal()
	actions := []ActionItem{
		{Key: "s", Title: "Stop Container"},
	}
	customCmds := []ActionItem{
		{Key: "1", Title: "Run Backup", Command: "docker exec -it c1 backup.sh"},
	}
	appActions := []ActionItem{
		{Key: "q", Title: "Quit"},
	}
	m.Show("Actions: my-container", actions, customCmds, appActions)

	// Select index 1 (Run Backup) to test command preview
	m.SelectedIndex = 1
	view := m.View()
	if !strings.Contains(view, "docker exec -it c1 backup.sh") {
		t.Errorf("View should contain command preview, got %q", view)
	}
	if !strings.Contains(view, "Actions: my-container") {
		t.Errorf("View should contain title, got %q", view)
	}
	if !strings.Contains(view, "Stop Container") {
		t.Errorf("View should contain item title, got %q", view)
	}
	if !strings.Contains(view, "[s]") {
		t.Errorf("View should contain item key badge, got %q", view)
	}
	if !strings.Contains(view, "ACTIONS") {
		t.Errorf("View should contain ACTIONS heading, got %q", view)
	}
	if !strings.Contains(view, "CUSTOM COMMANDS") {
		t.Errorf("View should contain CUSTOM COMMANDS heading, got %q", view)
	}
	if !strings.Contains(view, "APPLICATION") {
		t.Errorf("View should contain APPLICATION heading, got %q", view)
	}

	bg := "background-row1\nbackground-row2\nbackground-row3"
	overlay := m.Overlay(bg, 80, 24)
	if !strings.Contains(overlay, "Actions: my-container") {
		t.Errorf("Overlay should contain modal view, got %q", overlay)
	}
}

func TestActionMenuModal_FooterBindings(t *testing.T) {
	m := NewActionMenuModal()
	bindings := m.FooterBindings()
	if len(bindings) != 4 {
		t.Fatalf("FooterBindings should return 4 bindings, got %d", len(bindings))
	}
}

func TestActionMenuModal_CaseSensitiveShortcuts(t *testing.T) {
	m := NewActionMenuModal()
	actions := []ActionItem{
		{Key: "p", Title: "Pause", Action: func() tea.Msg { return testActionMsg{id: "pause"} }},
		{Key: "P", Title: "Prune", Action: func() tea.Msg { return testActionMsg{id: "prune"} }},
	}
	m.Show("Actions", actions, nil, nil)

	// Pressing 'P' must trigger Prune, NOT Pause
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'P', Text: "P"})
	if cmd == nil || cmd() != (testActionMsg{id: "prune"}) {
		t.Errorf("Expected 'prune' for 'P', got %v", cmd)
	}

	// Pressing 'p' must trigger Pause
	m.Show("Actions", actions, nil, nil)
	_, cmd = m.Update(tea.KeyPressMsg{Code: 'p', Text: "p"})
	if cmd == nil || cmd() != (testActionMsg{id: "pause"}) {
		t.Errorf("Expected 'pause' for 'p', got %v", cmd)
	}
}

func TestActionMenuModal_ScrollWithViewport(t *testing.T) {
	m := NewActionMenuModal()
	m.SetHeight(24)
	var actions []ActionItem
	for i := 1; i <= 20; i++ {
		actions = append(actions, ActionItem{
			Key:   string(rune('a' + i - 1)),
			Title: "Action Item " + string(rune('A'+i-1)),
		})
	}
	m.Show("Many Actions", actions, nil, nil)

	// Terminal height 24 rows
	viewConstrained := m.ViewWithHeight(24)
	lines := strings.Split(viewConstrained, "\n")
	if len(lines) > 24 {
		t.Errorf("Modal height %d exceeds maximum height 24", len(lines))
	}

	// Navigate down to item 18 (below the first page)
	for range 18 {
		m, _ = m.Update(tea.KeyPressMsg{Code: 'j'})
	}
	if m.SelectedIndex != 18 {
		t.Fatalf("SelectedIndex = %d, want 18", m.SelectedIndex)
	}
	if m.ScrollOffset == 0 {
		t.Fatalf("ScrollOffset should be > 0 after navigating to item 18, got %d", m.ScrollOffset)
	}

	// In the constrained view, selected item must be present
	viewWithSelected := m.ViewWithHeight(24)
	if !strings.Contains(viewWithSelected, "Action Item S") {
		t.Errorf("Selected item 'Action Item S' not visible in constrained view: %s", viewWithSelected)
	}

	idxBefore := m.SelectedIndex

	// Mouse wheel UP moves SelectedIndex up (matching tree behavior)
	m, _ = m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
	if m.SelectedIndex != idxBefore-1 {
		t.Errorf("SelectedIndex after wheel-up = %d, want %d", m.SelectedIndex, idxBefore-1)
	}

	// Mouse wheel DOWN moves SelectedIndex back down (matching tree behavior)
	m, _ = m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	if m.SelectedIndex != idxBefore {
		t.Errorf("SelectedIndex after wheel-down = %d, want %d", m.SelectedIndex, idxBefore)
	}

	// Scrolling up to the top should adjust ScrollOffset to 0 and show top items
	for range 18 {
		m, _ = m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelUp})
	}
	if m.SelectedIndex != 0 {
		t.Errorf("SelectedIndex after scrolling to top = %d, want 0", m.SelectedIndex)
	}
	if m.ScrollOffset != 0 {
		t.Errorf("ScrollOffset at top = %d, want 0", m.ScrollOffset)
	}
	topView := m.ViewWithHeight(24)
	if !strings.Contains(topView, "Action Item A") {
		t.Errorf("First item should be visible at top: %s", topView)
	}
}
