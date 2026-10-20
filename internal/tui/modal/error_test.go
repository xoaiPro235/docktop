package modal

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/xoaiPro235/docktop/internal/tui/theme"
)

func init() {
	theme.SetThemeByName("tokyo-night")
}

func TestNewErrorModal(t *testing.T) {
	m := NewErrorModal()
	if m.Active {
		t.Error("NewErrorModal should start inactive")
	}
}

func TestErrorModal_Show(t *testing.T) {
	m := NewErrorModal()

	// Show with explicit title
	m.Show("Custom Error", "Something went wrong")
	if !m.Active {
		t.Error("ErrorModal should be active after Show")
	}
	if m.Title != "Custom Error" || m.Message != "Something went wrong" {
		t.Errorf("Unexpected Title or Message: %q, %q", m.Title, m.Message)
	}

	// Show with default fallback title
	m.Show("", "Fallback title test")
	if m.Title != "Error" {
		t.Errorf("Empty title should default to 'Error', got %q", m.Title)
	}
}

func TestErrorModal_Hide(t *testing.T) {
	m := NewErrorModal()
	m.Show("Err", "Msg")
	m.Hide()
	if m.Active {
		t.Error("ErrorModal should be inactive after Hide")
	}
}

func TestErrorModal_Init(t *testing.T) {
	m := NewErrorModal()
	if cmd := m.Init(); cmd != nil {
		t.Errorf("ErrorModal.Init() = %v, want nil", cmd)
	}
}

func TestErrorModal_Update(t *testing.T) {
	// 1. Inactive modal ignores events
	m := NewErrorModal()
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.Active {
		t.Error("Inactive modal changed state")
	}

	// 2. Dismiss with Enter key
	m.Show("Err", "Msg")
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.Active {
		t.Error("Active modal should be dismissed on Enter")
	}

	// 3. Dismiss with Esc key
	m.Show("Err", "Msg")
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if m.Active {
		t.Error("Active modal should be dismissed on Esc")
	}

	// 4. Dismiss with mouse click
	m.Show("Err", "Msg")
	m, _ = m.Update(tea.MouseClickMsg{})
	if m.Active {
		t.Error("Active modal should be dismissed on MouseClick")
	}
}

func TestErrorModal_View(t *testing.T) {
	m := NewErrorModal()
	m.Show("Connection Failed", "Docker daemon is unreachable")
	view := m.View()
	if !strings.Contains(view, "Connection Failed") {
		t.Errorf("View should contain title 'Connection Failed', got %q", view)
	}
	if !strings.Contains(view, "Docker daemon is unreachable") {
		t.Errorf("View should contain message 'Docker daemon is unreachable', got %q", view)
	}
}

func TestErrorModal_Overlay(t *testing.T) {
	m := NewErrorModal()
	bg := "background-window-text"

	m.Show("Fatal Error", "Out of memory")
	overlay := m.Overlay(bg, 80, 24)
	if !strings.Contains(overlay, "Fatal Error") {
		t.Errorf("Active Overlay() should include modal content, got %q", overlay)
	}
}

func TestErrorModal_FooterBindings(t *testing.T) {
	m := NewErrorModal()
	bindings := m.FooterBindings()
	if len(bindings) == 0 {
		t.Error("FooterBindings should not be empty")
	}
	if !key.Matches(tea.KeyPressMsg{Code: tea.KeyEnter}, bindings[0]) {
		t.Error("FooterBindings[0] should match Enter key")
	}
}
