package container

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	mobyContainer "github.com/moby/moby/api/types/container"
	"github.com/xoaiPro235/docktop/internal/docker"
	"github.com/xoaiPro235/docktop/internal/docker/mocks"
	"github.com/xoaiPro235/docktop/internal/tui/modal"
)

func TestModel_AttachKeyRunsShellForRunningContainer(t *testing.T) {
	tests := []struct {
		name        string
		state       docker.ContainerState
		actionRun   bool
		wantCommand bool
		wantCall    bool
	}{
		{
			name:        "running container",
			state:       docker.StateRunning,
			wantCommand: true,
			wantCall:    true,
		},
		{
			name:  "stopped container",
			state: docker.StateExited,
		},
		{
			name:      "container action in progress",
			state:     docker.StateRunning,
			actionRun: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCli := &mocks.MockContainerService{
				AttachShellCmdFunc: func(containerID string) *exec.Cmd {
					return exec.Command("true")
				},
			}
			m := New(mockCli)
			m.updateContainers([]docker.ContainerInfo{
				{ID: "c1", Name: "web", State: tt.state},
			})
			if tt.actionRun {
				m.runningActions["c1"] = RunningAction{Action: modal.ActionStop}
			}

			_, cmd := m.Update(tea.KeyPressMsg{Code: 'a', Text: "a"})
			if (cmd != nil) != tt.wantCommand {
				t.Errorf("Attach key returned command = %v, want command %v", cmd != nil, tt.wantCommand)
			}

			calls := mockCli.AttachShellCmdCalls()
			if (len(calls) > 0) != tt.wantCall {
				t.Fatalf("AttachShellCmd called %d times, want called = %v", len(calls), tt.wantCall)
			}
			if tt.wantCall && calls[0].ContainerID != "c1" {
				t.Errorf("AttachShellCmd container ID = %q, want %q", calls[0].ContainerID, "c1")
			}
		})
	}
}

func TestModel_fetchInspectCmd(t *testing.T) {
	// Empty container ID returns nil
	mockCli := &mocks.MockContainerService{}
	m := New(mockCli)
	if cmd := m.fetchInspectCmd(""); cmd() != nil {
		t.Error("fetchInspectCmd(\"\") should return nil msg")
	}

	// Valid container ID
	mockCli = &mocks.MockContainerService{
		InspectContainerFunc: func(ctx context.Context, id string) (*mobyContainer.InspectResponse, error) {
			return &mobyContainer.InspectResponse{ID: id}, nil
		},
	}
	m = New(mockCli)
	cmd := m.fetchInspectCmd("cid-123")
	msg := cmd()
	loadedMsg, ok := msg.(InspectLoadedMsg)
	if !ok {
		t.Fatalf("Expected InspectLoadedMsg, got %T", msg)
	}
	if loadedMsg.ContainerID != "cid-123" {
		t.Errorf("ContainerID = %v, want cid-123", loadedMsg.ContainerID)
	}
}

func TestModel_fetchTopCmd(t *testing.T) {
	// Empty container ID returns nil
	mockCli := &mocks.MockContainerService{}
	m := New(mockCli)
	if cmd := m.fetchTopCmd(""); cmd() != nil {
		t.Error("fetchTopCmd(\"\") should return nil msg")
	}

	// Valid container ID
	mockCli = &mocks.MockContainerService{
		ContainerTopFunc: func(ctx context.Context, id string) (*mobyContainer.TopResponse, error) {
			return &mobyContainer.TopResponse{Titles: []string{"PID", "CMD"}}, nil
		},
	}
	m = New(mockCli)
	cmd := m.fetchTopCmd("cid-456")
	msg := cmd()
	topMsg, ok := msg.(TopLoadedMsg)
	if !ok {
		t.Fatalf("Expected TopLoadedMsg, got %T", msg)
	}
	if topMsg.ContainerID != "cid-456" {
		t.Errorf("ContainerID = %v, want cid-456", topMsg.ContainerID)
	}
}

func TestModel_LifecycleCmds(t *testing.T) {
	tests := []struct {
		name       string
		action     modal.Action
		setupMock  func(*mocks.MockContainerService, *bool)
		runCmd     func(Model) ContainerActionMsg
		wantAction modal.Action
		wantErr    bool
	}{
		{
			name:   "StartContainer",
			action: modal.ActionStart,
			setupMock: func(m *mocks.MockContainerService, called *bool) {
				m.StartContainerFunc = func(ctx context.Context, id string) error {
					*called = true
					return nil
				}
			},
			runCmd: func(m Model) ContainerActionMsg {
				return m.startContainerCmd("c1")().(ContainerActionMsg)
			},
			wantAction: modal.ActionStart,
			wantErr:    false,
		},
		{
			name:   "StopContainer",
			action: modal.ActionStop,
			setupMock: func(m *mocks.MockContainerService, called *bool) {
				m.StopContainerFunc = func(ctx context.Context, id string) error {
					*called = true
					return nil
				}
			},
			runCmd: func(m Model) ContainerActionMsg {
				return m.stopContainerCmd("c1")().(ContainerActionMsg)
			},
			wantAction: modal.ActionStop,
			wantErr:    false,
		},
		{
			name:   "RestartContainer",
			action: modal.ActionRestart,
			setupMock: func(m *mocks.MockContainerService, called *bool) {
				m.RestartContainerFunc = func(ctx context.Context, id string) error {
					*called = true
					return nil
				}
			},
			runCmd: func(m Model) ContainerActionMsg {
				return m.restartContainerCmd("c1")().(ContainerActionMsg)
			},
			wantAction: modal.ActionRestart,
			wantErr:    false,
		},
		{
			name:   "PauseContainer",
			action: modal.ActionPause,
			setupMock: func(m *mocks.MockContainerService, called *bool) {
				m.PauseContainerFunc = func(ctx context.Context, id string) error {
					*called = true
					return nil
				}
			},
			runCmd: func(m Model) ContainerActionMsg {
				return m.pauseContainerCmd("c1")().(ContainerActionMsg)
			},
			wantAction: modal.ActionPause,
			wantErr:    false,
		},
		{
			name:   "UnpauseContainer",
			action: modal.ActionUnpause,
			setupMock: func(m *mocks.MockContainerService, called *bool) {
				m.UnpauseContainerFunc = func(ctx context.Context, id string) error {
					*called = true
					return nil
				}
			},
			runCmd: func(m Model) ContainerActionMsg {
				return m.unpauseContainerCmd("c1")().(ContainerActionMsg)
			},
			wantAction: modal.ActionUnpause,
			wantErr:    false,
		},
		{
			name:   "RemoveContainer Normal",
			action: modal.ActionRemove,
			setupMock: func(m *mocks.MockContainerService, called *bool) {
				m.RemoveContainerFunc = func(ctx context.Context, id string, opts docker.RemoveContainerOpts) error {
					*called = true
					if opts.Force {
						return errors.New("expected normal remove")
					}
					return nil
				}
			},
			runCmd: func(m Model) ContainerActionMsg {
				return m.removeContainerCmd("c1", false)().(ContainerActionMsg)
			},
			wantAction: modal.ActionRemove,
			wantErr:    false,
		},
		{
			name:   "RemoveContainer Force",
			action: modal.ActionRemoveForce,
			setupMock: func(m *mocks.MockContainerService, called *bool) {
				m.RemoveContainerFunc = func(ctx context.Context, id string, opts docker.RemoveContainerOpts) error {
					*called = true
					if !opts.Force {
						return errors.New("expected force remove")
					}
					return nil
				}
			},
			runCmd: func(m Model) ContainerActionMsg {
				return m.removeContainerCmd("c1", true)().(ContainerActionMsg)
			},
			wantAction: modal.ActionRemoveForce,
			wantErr:    false,
		},
		{
			name:   "PruneContainers",
			action: modal.ActionPrune,
			setupMock: func(m *mocks.MockContainerService, called *bool) {
				m.PruneContainersFunc = func(ctx context.Context) (mobyContainer.PruneReport, error) {
					*called = true
					return mobyContainer.PruneReport{}, nil
				}
			},
			runCmd: func(m Model) ContainerActionMsg {
				return m.pruneContainersCmd()().(ContainerActionMsg)
			},
			wantAction: modal.ActionPrune,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			called := false
			mockCli := &mocks.MockContainerService{}
			tt.setupMock(mockCli, &called)

			m := New(mockCli)
			msg := tt.runCmd(m)

			if !called {
				t.Errorf("%s was not called on mock", tt.name)
			}
			if msg.Action != tt.wantAction {
				t.Errorf("Action = %v, want %v", msg.Action, tt.wantAction)
			}
			if (msg.Err != nil) != tt.wantErr {
				t.Errorf("Err = %v, wantErr = %v", msg.Err, tt.wantErr)
			}
		})
	}
}

func TestContainerOpenBrowserCommand(t *testing.T) {
	tests := []struct {
		name        string
		container   docker.ContainerInfo
		actionRun   bool
		wantError   bool
		wantTitle   string
		wantMessage string
	}{
		{
			name: "running container without published ports",
			container: docker.ContainerInfo{
				ID: "ctr-no-ports", Name: "internal-worker", State: docker.StateRunning,
			},
			wantError:   true,
			wantTitle:   "No Public Port",
			wantMessage: `Container "internal-worker" has no ports mapped to the host.`,
		},
		{
			name: "stopped container with published port",
			container: docker.ContainerInfo{
				ID: "ctr-stopped", Name: "stopped-web", State: docker.StateExited,
				Ports: "0.0.0.0:8080->80/tcp",
			},
		},
		{
			name: "container action in progress",
			container: docker.ContainerInfo{
				ID: "ctr-busy", Name: "busy-web", State: docker.StateRunning,
				Ports: "0.0.0.0:8080->80/tcp",
			},
			actionRun: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(&mocks.MockContainerService{})
			m.updateContainers([]docker.ContainerInfo{tt.container})
			if tt.actionRun {
				m.runningActions[tt.container.ID] = RunningAction{Action: modal.ActionStop}
			}

			_, cmd := m.Update(tea.KeyPressMsg{Code: 'o', Text: "o"})
			if !tt.wantError {
				if cmd != nil {
					t.Fatalf("Open Browser returned unexpected command: %v", cmd)
				}
				return
			}

			if cmd == nil {
				t.Fatal("expected command reporting that no port is published")
			}
			result := cmd()
			errMsg, ok := result.(modal.ShowErrorMsg)
			if !ok {
				t.Fatalf("Open Browser command returned %T, want modal.ShowErrorMsg", result)
			}
			if errMsg.Title != tt.wantTitle {
				t.Errorf("error title = %q, want %q", errMsg.Title, tt.wantTitle)
			}
			if !strings.Contains(errMsg.Message, tt.wantMessage) {
				t.Errorf("error message = %q, want it to contain %q", errMsg.Message, tt.wantMessage)
			}
		})
	}
}
