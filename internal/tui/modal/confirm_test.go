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

func TestAction_Style(t *testing.T) {
	tests := []struct {
		name string
		a    Action
		want Style
	}{
		{"Stop", ActionStop, StyleDanger},
		{"Remove", ActionRemove, StyleDanger},
		{"RemoveForce", ActionRemoveForce, StyleDanger},
		{"Delete", ActionDelete, StyleDanger},
		{"Restart", ActionRestart, StyleWarning},
		{"Pause", ActionPause, StyleWarning},
		{"Prune", ActionPrune, StyleWarning},
		{"Start", ActionStart, StyleSuccess},
		{"Unpause", ActionUnpause, StyleSuccess},
		{"None", ActionNone, StyleDefault},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.a.Style(); got != tt.want {
				t.Errorf("Action.Style() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAction_String(t *testing.T) {
	tests := []struct {
		a    Action
		want string
	}{
		{ActionStart, "start"},
		{ActionStop, "stop"},
		{ActionRestart, "restart"},
		{ActionPause, "pause"},
		{ActionUnpause, "unpause"},
		{ActionRemove, "remove"},
		{ActionRemoveForce, "remove_force"},
		{ActionPrune, "prune"},
		{ActionDelete, "delete"},
		{ActionNone, ""},
	}
	for _, tt := range tests {
		if got := tt.a.String(); got != tt.want {
			t.Errorf("Action.String() for %v = %v, want %v", tt.a, got, tt.want)
		}
	}
}

func TestAction_Title(t *testing.T) {
	tests := []struct {
		a    Action
		want string
	}{
		{ActionStart, "Start"},
		{ActionStop, "Stop"},
		{ActionRestart, "Restart"},
		{ActionPause, "Pause"},
		{ActionUnpause, "Unpause"},
		{ActionRemove, "Remove"},
		{ActionRemoveForce, "Force Remove"},
		{ActionPrune, "Prune"},
		{ActionDelete, "Delete"},
		{ActionNone, ""},
	}
	for _, tt := range tests {
		if got := tt.a.Title(); got != tt.want {
			t.Errorf("Action.Title() for %v = %v, want %v", tt.a, got, tt.want)
		}
	}
}

func TestActionState_Lifecycle(t *testing.T) {
	state := ActionState{}
	if state.IsRunning() {
		t.Error("Initial ActionState should not be running")
	}

	state.Start(ActionRestart, "cid123", "my-web", "Restarting container")
	if !state.IsRunning() {
		t.Error("ActionState should be running after Start")
	}
	if state.Action != ActionRestart || state.TargetID != "cid123" || state.TargetName != "my-web" {
		t.Errorf("Unexpected ActionState after Start: %+v", state)
	}

	state.Finish("Successfully restarted")
	if state.IsRunning() {
		t.Error("ActionState should not be running after Finish")
	}
	if state.StatusMsg != "Successfully restarted" {
		t.Errorf("StatusMsg = %q, want 'Successfully restarted'", state.StatusMsg)
	}
	if state.Action != ActionNone {
		t.Errorf("Action = %v, want ActionNone", state.Action)
	}
}

func TestNewConfirmModal(t *testing.T) {
	m := NewConfirmModal()
	if m.Active {
		t.Error("New modal should be inactive")
	}
	if !m.SelectedYes {
		t.Error("New modal should have SelectedYes default to true")
	}
}

func TestConfirmModal_ShowAndHide(t *testing.T) {
	m := NewConfirmModal()
	m.Show("Are you sure?", "docker rm test", StyleDanger, ActionRemove, "id1", "test")

	if !m.Active {
		t.Error("Modal should be active after Show")
	}
	if m.Question != "Are you sure?" || m.Command != "docker rm test" {
		t.Errorf("Unexpected question or command: %q, %q", m.Question, m.Command)
	}

	m.Hide()
	if m.Active {
		t.Error("Modal should be inactive after Hide")
	}
	if m.Confirmed {
		t.Error("Modal Confirmed should be false after Hide")
	}
}

func TestConfirmModal_ShowActions(t *testing.T) {
	m := NewConfirmModal()

	// Container actions
	containerActions := []Action{
		ActionStart, ActionStop, ActionPause, ActionUnpause,
		ActionRestart, ActionRemove, ActionRemoveForce, ActionPrune,
	}
	for _, act := range containerActions {
		m.ShowContainerAction(act, "cid1", "cname1")
		if !m.Active || m.Action != act {
			t.Errorf("ShowContainerAction failed for action %v", act)
		}
	}

	// Image actions
	m.ShowImageAction(ActionRemove, "img1", "alpine:latest")
	if !m.Active || m.Action != ActionRemove {
		t.Errorf("ShowImageAction Remove failed")
	}
	m.ShowImageAction(ActionPrune, "img1", "alpine:latest")
	if !m.Active || m.Action != ActionPrune {
		t.Errorf("ShowImageAction Prune failed")
	}

	// Volume actions
	m.ShowVolumeAction(ActionRemove, "my-vol")
	if !m.Active || m.Action != ActionRemove {
		t.Errorf("ShowVolumeAction Remove failed")
	}
	m.ShowVolumeAction(ActionPrune, "my-vol")
	if !m.Active || m.Action != ActionPrune {
		t.Errorf("ShowVolumeAction Prune failed")
	}

	// Network actions
	m.ShowNetworkAction(ActionDelete, "net1", "bridge")
	if !m.Active || m.Action != ActionDelete {
		t.Errorf("ShowNetworkAction Delete failed")
	}
	m.ShowNetworkAction(ActionPrune, "net1", "bridge")
	if !m.Active || m.Action != ActionPrune {
		t.Errorf("ShowNetworkAction Prune failed")
	}
}

func TestConfirmModal_Init(t *testing.T) {
	m := NewConfirmModal()
	if cmd := m.Init(); cmd != nil {
		t.Errorf("ConfirmModal.Init() = %v, want nil", cmd)
	}
}

func TestConfirmModal_Update(t *testing.T) {
	// 1. Inactive modal ignores keys
	m := NewConfirmModal()
	m, _ = m.Update(tea.KeyPressMsg{Code: 'y'})
	if m.Active || m.Confirmed {
		t.Error("Inactive modal responded to keypress")
	}

	// 2. Press Yes ('y')
	m.Show("Confirm?", "cmd", StyleDanger, ActionRemove, "1", "name")
	m, _ = m.Update(tea.KeyPressMsg{Code: 'y'})
	if m.Active {
		t.Error("Modal should become inactive after pressing Yes")
	}
	if !m.Confirmed {
		t.Error("Modal should be Confirmed after pressing Yes")
	}

	// 3. Press No ('n')
	m.Show("Confirm?", "cmd", StyleDanger, ActionRemove, "1", "name")
	m, _ = m.Update(tea.KeyPressMsg{Code: 'n'})
	if m.Active {
		t.Error("Modal should become inactive after pressing No")
	}
	if m.Confirmed {
		t.Error("Modal should NOT be Confirmed after pressing No")
	}

	// 4. Press Escape
	m.Show("Confirm?", "cmd", StyleDanger, ActionRemove, "1", "name")
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if m.Active || m.Confirmed {
		t.Error("Modal should be dismissed without confirm on Esc")
	}

	// 5. Toggle switch (Tab)
	m.Show("Confirm?", "cmd", StyleDanger, ActionRemove, "1", "name")
	if !m.SelectedYes {
		t.Error("SelectedYes should be true initially")
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if m.SelectedYes {
		t.Error("SelectedYes should toggle to false after Tab")
	}
	// Press Enter when No is selected -> Confirmed should be false
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if m.Active {
		t.Error("Modal should be inactive after Enter")
	}
	if m.Confirmed {
		t.Error("Modal should not be confirmed when Enter pressed with SelectedYes=false")
	}
}

func TestConfirmModal_ViewAndOverlay(t *testing.T) {
	m := NewConfirmModal()
	m.Show("Are you sure you want to delete this?", "docker rm test", StyleDanger, ActionRemove, "1", "test")

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
