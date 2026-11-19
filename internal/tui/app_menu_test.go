package tui

import (
	"reflect"
	"runtime"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/xoaiPro235/docktop/internal/config"
	"github.com/xoaiPro235/docktop/internal/docker"
	"github.com/xoaiPro235/docktop/internal/tui/modal"
)

func TestAppModel_ActionMenu_OpenAndDismiss(t *testing.T) {
	mockCli := helperMockService()

	cfg := &config.Config{
		CustomCommands: config.CustomCommandsConfig{
			Containers: []config.CustomCommand{
				{Name: "Shell Bash", Command: "docker exec -it {{ .ID }} bash"},
			},
		},
	}

	app := NewAppWithConfig(mockCli, cfg)
	app.width = 100
	app.height = 30
	app.ready = true

	// Populate container
	app.containerModel.SetContainers([]docker.ContainerInfo{
		{ID: "c1", Name: "app-prod", State: docker.StateRunning},
	})

	// 1. Menu initially inactive
	if app.actionMenuModal.Active {
		t.Error("Action menu should be inactive initially")
	}

	// 2. Press 'x' to open menu
	updatedApp, cmd := app.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	app = updatedApp.(AppModel)
	if cmd != nil {
		_ = cmd()
	}

	if !app.actionMenuModal.Active {
		t.Fatal("Action menu should become active after pressing 'x'")
	}
	if !strings.Contains(app.actionMenuModal.Title, "app-prod") {
		t.Errorf("Menu title = %q, expected to contain container name", app.actionMenuModal.Title)
	}

	// Verify menu contains both default actions and custom commands
	foundCustom := false
	foundStop := false
	for _, item := range app.actionMenuModal.Items() {
		if item.Title == "Shell Bash" {
			foundCustom = true
		}
		if item.Title == "Stop" {
			foundStop = true
		}
	}
	if !foundCustom {
		t.Error("Action menu items missing custom command 'Shell Bash'")
	}
	if !foundStop {
		t.Error("Action menu items missing container action 'Stop'")
	}

	if len(app.actionMenuModal.CustomCmds) == 0 {
		t.Error("Action menu missing custom commands")
	} else if app.actionMenuModal.CustomCmds[0].Command != "docker exec -it c1 bash" {
		t.Errorf("Custom command preview = %q, want 'docker exec -it c1 bash'", app.actionMenuModal.CustomCmds[0].Command)
	}

	// Verify View contains the overlay
	viewStr := app.View().Content
	if !strings.Contains(viewStr, "Shell Bash") {
		t.Errorf("App view does not contain action menu overlay: %s", viewStr)
	}

	// 3. Press 'x' or 'Esc' to dismiss
	updatedApp, _ = app.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	app = updatedApp.(AppModel)
	if app.actionMenuModal.Active {
		t.Error("Action menu should be dismissed after Esc")
	}
}

func TestAppModel_ActionMenu_CustomCommandConfirmFlow(t *testing.T) {
	mockCli := helperMockService()

	cfg := &config.Config{
		CustomCommands: config.CustomCommandsConfig{
			Containers: []config.CustomCommand{
				{Name: "Inspect IP", Command: "docker inspect {{ .ID }}"},
			},
		},
	}

	app := NewAppWithConfig(mockCli, cfg)
	app.width = 100
	app.height = 30
	app.ready = true

	app.containerModel.SetContainers([]docker.ContainerInfo{
		{ID: "c12345", Name: "my-web", State: docker.StateRunning},
	})

	// Open menu
	updatedApp, _ := app.Update(tea.KeyPressMsg{Code: 'x', Text: "x"})
	app = updatedApp.(AppModel)

	// Find the custom command item
	var customAction *modal.ActionItem
	for _, item := range app.actionMenuModal.Items() {
		if item.Title == "Inspect IP" {
			itemCopy := item
			customAction = &itemCopy
			break
		}
	}

	if customAction == nil {
		t.Fatal("Could not find 'Inspect IP' action item in menu")
	}

	// Trigger custom command action
	resMsg := customAction.Action()
	confirmMsg, ok := resMsg.(modal.ShowConfirmMsg)
	if !ok {
		t.Fatalf("Expected modal.ShowConfirmMsg, got %T", resMsg)
	}

	if !strings.Contains(confirmMsg.Question, "Inspect IP") {
		t.Errorf("Question = %q, expected to contain command name", confirmMsg.Question)
	}
	if confirmMsg.Command != "docker inspect c12345" {
		t.Errorf("Command = %q, want 'docker inspect c12345'", confirmMsg.Command)
	}
	if confirmMsg.OnConfirm == nil {
		t.Error("OnConfirm should not be nil")
	}
}

type sampleContainer struct {
	ID    string
	Name  string
	Image string
}

func TestRenderCustomCommand(t *testing.T) {
	c := sampleContainer{
		ID:    "abc12345",
		Name:  "web-server",
		Image: "nginx:alpine",
	}

	tests := []struct {
		name     string
		template string
		data     any
		want     string
		wantErr  bool
	}{
		{
			name:     "valid container id and name substitution",
			template: "docker exec -it {{ .ID }} sh -c 'echo {{ .Name }}'",
			data:     c,
			want:     "docker exec -it abc12345 sh -c 'echo web-server'",
			wantErr:  false,
		},
		{
			name:     "simple inspect command",
			template: "docker inspect {{ .ID }}",
			data:     c,
			want:     "docker inspect abc12345",
			wantErr:  false,
		},
		{
			name:     "template syntax error",
			template: "docker run {{ .ID",
			data:     c,
			want:     "",
			wantErr:  true,
		},
		{
			name:     "template missing field error",
			template: "docker inspect {{ .NonExistentField }}",
			data:     c,
			want:     "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := RenderCustomCommand(tt.template, tt.data)
			if (err != nil) != tt.wantErr {
				t.Fatalf("RenderCustomCommand() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("RenderCustomCommand() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildCustomCommandExec(t *testing.T) {
	t.Run("empty command returns nil", func(t *testing.T) {
		if got := BuildCustomCommandExec("   ", false); got != nil {
			t.Errorf("Expected nil for empty command, got %v", got)
		}
		if got := BuildCustomCommandExec("", true); got != nil {
			t.Errorf("Expected nil for empty command with pause, got %v", got)
		}
	})

	t.Run("standard command without pause", func(t *testing.T) {
		cmd := BuildCustomCommandExec("docker history abc123", false)
		if cmd == nil {
			t.Fatal("Expected non-nil *exec.Cmd")
		}
		wantArgs := []string{"docker", "history", "abc123"}
		if !reflect.DeepEqual(cmd.Args, wantArgs) {
			t.Errorf("cmd.Args = %v, want %v", cmd.Args, wantArgs)
		}
	})

	t.Run("command with pause", func(t *testing.T) {
		cmd := BuildCustomCommandExec("docker inspect abc123", true)
		if cmd == nil {
			t.Fatal("Expected non-nil *exec.Cmd")
		}

		if runtime.GOOS == "windows" {
			wantArgs := []string{"cmd.exe", "/c", "docker inspect abc123 & echo. & pause"}
			if !reflect.DeepEqual(cmd.Args, wantArgs) {
				t.Errorf("cmd.Args = %v, want %v", cmd.Args, wantArgs)
			}
		} else {
			wantArgs := []string{"sh", "-c", "docker inspect abc123; printf \"\\n\\033[1;36m[Press Enter to return to docktop]\\033[0m \"; read -r _"}
			if !reflect.DeepEqual(cmd.Args, wantArgs) {
				t.Errorf("cmd.Args = %v, want %v", cmd.Args, wantArgs)
			}
		}
	})
}

func TestExecCustomCommand(t *testing.T) {
	if cmd := ExecCustomCommand("", false); cmd != nil {
		t.Error("Expected nil for empty command")
	}

	cmd := ExecCustomCommand("echo hello", false)
	if cmd == nil {
		t.Fatal("ExecCustomCommand returned nil")
	}

	cmdWithPause := ExecCustomCommand("echo hello", true)
	if cmdWithPause == nil {
		t.Fatal("ExecCustomCommand with pause returned nil")
	}
}
