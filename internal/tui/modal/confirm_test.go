package modal

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/xoaiPro235/docktop/internal/tui/theme"
)

func init() {
	theme.SetThemeByName("tokyo-night")
}

func TestConfirmModal_Init(t *testing.T) {
	m := NewConfirmModal()
	if cmd := m.Init(); cmd != nil {
		t.Errorf("ConfirmModal.Init() = %v, want nil", cmd)
	}
}

type testConfirmMsg struct{}

func TestConfirmModal_Update(t *testing.T) {
	// 1. Inactive modal ignores keys
	m := NewConfirmModal()
	m, cmd := m.Update(tea.KeyPressMsg{Code: 'y'})
	if m.Active || cmd != nil {
		t.Error("Inactive modal responded to keypress")
	}

	testCmd := func() tea.Msg { return testConfirmMsg{} }

	// 2. Press Yes ('y')
	m.Show("Confirm?", "cmd", StyleDanger, testCmd)
	m, cmd = m.Update(tea.KeyPressMsg{Code: 'y'})
	if m.Active {
		t.Error("Modal should become inactive after pressing Yes")
	}
	if cmd == nil || cmd() != (testConfirmMsg{}) {
		t.Error("Modal should return OnConfirm cmd after pressing Yes")
	}

	// 3. Press No ('n')
	m.Show("Confirm?", "cmd", StyleDanger, testCmd)
	m, cmd = m.Update(tea.KeyPressMsg{Code: 'n'})
	if m.Active {
		t.Error("Modal should become inactive after pressing No")
	}
	if cmd != nil {
		t.Error("Modal should NOT return cmd after pressing No")
	}

	// 4. Press Escape
	m.Show("Confirm?", "cmd", StyleDanger, testCmd)
	m, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if m.Active || cmd != nil {
		t.Error("Modal should be dismissed without cmd on Esc")
	}

	// 5. Toggle switch (Tab)
	m.Show("Confirm?", "cmd", StyleDanger, testCmd)
	if !m.SelectedYes {
		t.Error("SelectedYes should be true initially")
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if m.SelectedYes {
		t.Error("SelectedYes should toggle to false after Tab")
	}
	// Press Enter when No is selected -> cmd should be nil
	m, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.Active {
		t.Error("Modal should be inactive after Enter")
	}
	if cmd != nil {
		t.Error("Modal should not execute cmd when Enter pressed with SelectedYes=false")
	}

	// 6. Press Enter with Yes selected -> cmd should execute
	m.Show("Confirm?", "cmd", StyleDanger, testCmd)
	m, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.Active {
		t.Error("Modal should be inactive after Enter")
	}
	if cmd == nil || cmd() != (testConfirmMsg{}) {
		t.Error("Modal should execute cmd when Enter pressed with SelectedYes=true")
	}
}

func TestConfirmModal_ViewAndOverlay(t *testing.T) {
	m := NewConfirmModal()
	m.Show("Are you sure you want to delete this?", "docker rm test", StyleDanger, nil)

	view := m.View()
	if !strings.Contains(view, "docker rm test") {
		t.Errorf("View should contain command preview, got %q", view)
	}
	if !strings.Contains(view, "Are you sure you want to delete this?") {
		t.Errorf("View should contain question, got %q", view)
	}

	bg := "background-content\nbackground-line2"
	overlay := m.Overlay(bg, 80, 24)
	if !strings.Contains(overlay, "docker rm test") {
		t.Errorf("Overlay should contain modal contents, got %q", overlay)
	}
}

func TestConfirmModal_FooterBindings(t *testing.T) {
	m := NewConfirmModal()
	bindings := m.FooterBindings()
	if len(bindings) != 3 {
		t.Fatalf("FooterBindings should return 3 bindings, got %d", len(bindings))
	}
	if bindings[0].Help().Key == "" {
		t.Error("First binding help key should not be empty")
	}
}
