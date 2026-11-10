package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/moby/moby/api/types/events"
	"github.com/xoaiPro235/docktop/internal/docker"
	"github.com/xoaiPro235/docktop/internal/docker/mocks"
	"github.com/xoaiPro235/docktop/internal/tui/components/container"
	"github.com/xoaiPro235/docktop/internal/tui/theme"
)

func init() {
	theme.SetThemeByName("tokyo-night")
}

func helperMockService() *mocks.MockService {
	eventCh := make(chan events.Message)
	return &mocks.MockService{
		HostFunc: func() string {
			return "test-docker-host"
		},
		StreamEventsFunc: func(ctx context.Context) (<-chan events.Message, <-chan error) {
			return eventCh, nil
		},
		ListContainersFunc: func(ctx context.Context) ([]docker.ContainerInfo, error) {
			return []docker.ContainerInfo{{ID: "c1", Name: "app"}}, nil
		},
		ListImagesFunc: func(ctx context.Context) ([]docker.ImageInfo, error) {
			return []docker.ImageInfo{{ID: "i1", Tag: "alpine:latest"}}, nil
		},
		ListVolumesFunc: func(ctx context.Context) ([]docker.VolumeInfo, error) {
			return []docker.VolumeInfo{}, nil
		},
		ListNetworksFunc: func(ctx context.Context) ([]docker.NetworkInfo, error) {
			return []docker.NetworkInfo{}, nil
		},
	}
}

func Test_waitForDockerEvent(t *testing.T) {
	// 1. Nil channel signals that the event stream is closed
	cmdNil := waitForDockerEvent(nil, nil)
	if msg := cmdNil(); msg != (DockerEventStreamClosedMsg{}) {
		t.Errorf("waitForDockerEvent(nil)() = %v, want DockerEventStreamClosedMsg", msg)
	}

	// 2. Active channel with event
	ch := make(chan events.Message, 1)
	ch <- events.Message{Action: "start", Type: "container"}
	cmd := waitForDockerEvent(ch, nil)
	msg := cmd()
	dockerMsg, ok := msg.(DockerEventMsg)
	if !ok || dockerMsg.Action != "start" {
		t.Errorf("Expected DockerEventMsg with action start, got %v", msg)
	}

	// 3. Closed channel signals that the event stream is closed
	close(ch)
	cmdClosed := waitForDockerEvent(ch, nil)
	if msg := cmdClosed(); msg != (DockerEventStreamClosedMsg{}) {
		t.Errorf("waitForDockerEvent(closed)() = %v, want DockerEventStreamClosedMsg", msg)
	}

	// 4. An error from the event stream signals that it is closed
	errCh := make(chan error, 1)
	errCh <- context.DeadlineExceeded
	cmdError := waitForDockerEvent(nil, errCh)
	if msg := cmdError(); msg != (DockerEventStreamClosedMsg{}) {
		t.Errorf("waitForDockerEvent(error)() = %v, want DockerEventStreamClosedMsg", msg)
	}
}

func TestNewApp(t *testing.T) {
	mockSvc := helperMockService()
	app := NewApp(mockSvc)

	if app.dockerClient == nil {
		t.Error("dockerClient should be assigned")
	}
	if app.activeTab != TabContainers {
		t.Errorf("activeTab = %v, want TabContainers", app.activeTab)
	}
	if !app.daemonConnected {
		t.Error("daemonConnected = false, want true")
	}
	if app.reconnecting {
		t.Error("reconnecting = true, want false")
	}
	if app.reconnectAttempts != 0 {
		t.Errorf("reconnectAttempts = %d, want 0", app.reconnectAttempts)
	}
	if app.isAnyModalActive() {
		t.Error("No modal should be active initially")
	}
}

func TestAppModel_ManualReconnect(t *testing.T) {
	tests := []struct {
		name        string
		pingErr     error
		wantModal   bool
		wantOnline  bool
		wantMessage string
	}{
		{
			name:       "success",
			wantOnline: true,
		},
		{
			name:        "failure",
			pingErr:     errors.New("daemon unavailable"),
			wantModal:   true,
			wantMessage: "daemon unavailable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockSvc := helperMockService()
			mockSvc.PingFunc = func() error { return tt.pingErr }
			app := NewApp(mockSvc)

			model, cmd := app.Update(tea.KeyPressMsg{Code: 'r', Mod: tea.ModCtrl})
			app = model.(AppModel)
			if !app.reconnecting || app.daemonConnected {
				t.Fatalf("after Ctrl+R: reconnecting=%v, daemonConnected=%v; want true, false",
					app.reconnecting, app.daemonConnected)
			}
			if app.reconnectAttempts != 0 {
				t.Errorf("reconnectAttempts = %d, want 0 for manual reconnect", app.reconnectAttempts)
			}
			if cmd == nil {
				t.Fatal("Ctrl+R should return a reconnect command")
			}

			msg := cmd()
			result, ok := msg.(DockerManualReconnectResultMsg)
			if !ok {
				t.Fatalf("reconnect command returned %T, want DockerManualReconnectResultMsg", msg)
			}
			model, _ = app.Update(result)
			app = model.(AppModel)

			if app.daemonConnected != tt.wantOnline {
				t.Errorf("daemonConnected = %v, want %v", app.daemonConnected, tt.wantOnline)
			}
			if app.reconnecting {
				t.Error("reconnecting should be false after the manual attempt completes")
			}
			if app.errorModal.Active != tt.wantModal {
				t.Errorf("errorModal.Active = %v, want %v", app.errorModal.Active, tt.wantModal)
			}
			if tt.wantMessage != "" && !strings.Contains(app.errorModal.Message, tt.wantMessage) {
				t.Errorf("error modal message %q does not contain %q", app.errorModal.Message, tt.wantMessage)
			}
			if tt.wantOnline && len(mockSvc.StreamEventsCalls()) != 2 {
				t.Errorf("StreamEvents called %d times, want 2 after reconnect", len(mockSvc.StreamEventsCalls()))
			}
		})
	}
}

func TestAppModel_AutomaticReconnectRetriesAreLimited(t *testing.T) {
	mockSvc := helperMockService()
	mockSvc.PingFunc = func() error { return errors.New("daemon unavailable") }
	app := NewApp(mockSvc)
	app.reconnecting = true
	app.reconnectAttempts = maxReconnectAttempts

	model, cmd := app.Update(DockerReconnectTickMsg{})
	app = model.(AppModel)

	if app.reconnecting {
		t.Error("reconnecting should stop after maxReconnectAttempts is exceeded")
	}
	if app.daemonConnected {
		t.Error("daemonConnected should remain false after retries are exhausted")
	}
	if app.reconnectAttempts != 0 {
		t.Errorf("reconnectAttempts = %d, want 0 after retries are exhausted", app.reconnectAttempts)
	}
	if cmd != nil {
		t.Error("no further retry command should be scheduled after retries are exhausted")
	}
	if got := len(mockSvc.PingCalls()); got != 1 {
		t.Errorf("Ping called %d times, want 1", got)
	}
}

func TestAppModel_AutomaticReconnectAfterEventStreamCloses(t *testing.T) {
	mockSvc := helperMockService()
	mockSvc.PingFunc = func() error { return nil }
	app := NewApp(mockSvc)

	model, cmd := app.Update(DockerEventStreamClosedMsg{})
	app = model.(AppModel)
	if app.daemonConnected || !app.reconnecting {
		t.Fatalf("after event stream closes: daemonConnected=%v, reconnecting=%v; want false, true",
			app.daemonConnected, app.reconnecting)
	}
	if app.reconnectAttempts != 1 {
		t.Errorf("reconnectAttempts = %d, want 1", app.reconnectAttempts)
	}
	if cmd == nil {
		t.Fatal("event stream closure should schedule a reconnect tick")
	}

	model, cmd = app.Update(DockerReconnectTickMsg{})
	app = model.(AppModel)
	if !app.daemonConnected || app.reconnecting {
		t.Errorf("after successful retry: daemonConnected=%v, reconnecting=%v; want true, false",
			app.daemonConnected, app.reconnecting)
	}
	if app.reconnectAttempts != 0 {
		t.Errorf("reconnectAttempts = %d, want 0 after successful retry", app.reconnectAttempts)
	}
	if cmd == nil {
		t.Error("successful retry should restart event listening and refresh data")
	}
	if got := len(mockSvc.StreamEventsCalls()); got != 2 {
		t.Errorf("StreamEvents called %d times, want 2", got)
	}
}

func TestAppModel_FooterBindingsShowsReconnectWhenDisconnected(t *testing.T) {
	app := NewApp(helperMockService())
	app.daemonConnected = false

	hasReconnect := func(bindings []key.Binding) bool {
		for _, binding := range bindings {
			if binding.Help().Key == "ctrl+r" {
				return true
			}
		}
		return false
	}

	if !hasReconnect(app.FooterBindings()) {
		t.Fatal("FooterBindings should include Ctrl+R when disconnected")
	}

	app.daemonConnected = true
	if hasReconnect(app.FooterBindings()) {
		t.Error("FooterBindings should not include Ctrl+R when connected")
	}
}

func TestAppModel_Update_AttachFinished(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		wantModal bool
	}{
		{name: "success"},
		{name: "failure", err: errors.New("attach failed"), wantModal: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApp(helperMockService())
			model, _ := app.Update(container.AttachFinishedMsg{Err: tt.err})
			app = model.(AppModel)

			if app.errorModal.Active != tt.wantModal {
				t.Fatalf("error modal active = %v, want %v", app.errorModal.Active, tt.wantModal)
			}
			if tt.wantModal {
				if app.errorModal.Title != "Container Attach Failed" {
					t.Errorf("error modal title = %q, want %q", app.errorModal.Title, "Container Attach Failed")
				}
				if !strings.Contains(app.errorModal.Message, tt.err.Error()) {
					t.Errorf("error modal message %q does not contain %q", app.errorModal.Message, tt.err)
				}
			}
		})
	}
}

func TestAppModel_FetchCommands(t *testing.T) {
	mockSvc := helperMockService()
	app := NewApp(mockSvc)

	// Fetch Containers
	cmdCtr := app.fetchContainersCmd()
	msgCtr := cmdCtr()
	if ctrList, ok := msgCtr.(ContainerListMsg); !ok || len(ctrList) != 1 {
		t.Errorf("fetchContainersCmd result = %v", msgCtr)
	}

	// Fetch Images
	cmdImg := app.fetchImagesCmd()
	msgImg := cmdImg()
	if imgList, ok := msgImg.(ImageListMsg); !ok || len(imgList) != 1 {
		t.Errorf("fetchImagesCmd result = %v", msgImg)
	}

	// Fetch Volumes
	cmdVol := app.fetchVolumesCmd()
	msgVol := cmdVol()
	if _, ok := msgVol.(VolumeListMsg); !ok {
		t.Errorf("fetchVolumesCmd result = %v", msgVol)
	}

	// Fetch Networks
	cmdNet := app.fetchNetworksCmd()
	msgNet := cmdNet()
	if _, ok := msgNet.(NetworkListMsg); !ok {
		t.Errorf("fetchNetworksCmd result = %v", msgNet)
	}
}

func TestAppModel_SwitchTab(t *testing.T) {
	mockSvc := helperMockService()
	app := NewApp(mockSvc)

	app.switchTab(TabImages)
	if app.activeTab != TabImages {
		t.Errorf("activeTab = %v, want TabImages", app.activeTab)
	}

	app.switchTab(TabVolumes)
	if app.activeTab != TabVolumes {
		t.Errorf("activeTab = %v, want TabVolumes", app.activeTab)
	}

	app.switchTab(TabNetworks)
	if app.activeTab != TabNetworks {
		t.Errorf("activeTab = %v, want TabNetworks", app.activeTab)
	}
}

func TestAppModel_Init(t *testing.T) {
	mockSvc := helperMockService()
	app := NewApp(mockSvc)

	cmd := app.Init()
	if cmd == nil {
		t.Error("Init() should return batch cmd")
	}
}

func TestAppModel_Update_WindowSize(t *testing.T) {
	mockSvc := helperMockService()
	app := NewApp(mockSvc)

	newModel, _ := app.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	updatedApp := newModel.(AppModel)

	if !updatedApp.ready {
		t.Error("ready should be true after WindowSizeMsg")
	}
	if updatedApp.width != 160 || updatedApp.height != 50 {
		t.Errorf("Dimensions = (%d, %d), want (160, 50)", updatedApp.width, updatedApp.height)
	}
}

func TestAppModel_Update_TabKeypresses(t *testing.T) {
	mockSvc := helperMockService()
	app := NewApp(mockSvc)
	app.ready = true

	// Key '2' -> TabImages
	newModel, _ := app.Update(tea.KeyPressMsg{Code: '2', Text: "2"})
	app = newModel.(AppModel)
	if app.activeTab != TabImages {
		t.Errorf("activeTab after '2' = %v, want TabImages", app.activeTab)
	}

	// Key '3' -> TabVolumes
	newModel, _ = app.Update(tea.KeyPressMsg{Code: '3', Text: "3"})
	app = newModel.(AppModel)
	if app.activeTab != TabVolumes {
		t.Errorf("activeTab after '3' = %v, want TabVolumes", app.activeTab)
	}

	// Key '4' -> TabNetworks
	newModel, _ = app.Update(tea.KeyPressMsg{Code: '4', Text: "4"})
	app = newModel.(AppModel)
	if app.activeTab != TabNetworks {
		t.Errorf("activeTab after '4' = %v, want TabNetworks", app.activeTab)
	}

	// Key '1' -> TabContainers
	newModel, _ = app.Update(tea.KeyPressMsg{Code: '1', Text: "1"})
	app = newModel.(AppModel)
	if app.activeTab != TabContainers {
		t.Errorf("activeTab after '1' = %v, want TabContainers", app.activeTab)
	}
}

func TestAppModel_IsAnySearching(t *testing.T) {
	tests := []struct {
		name  string
		start func(*AppModel)
	}{
		{
			name: "containers",
			start: func(app *AppModel) {
				app.containerModel, _ = app.containerModel.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
			},
		},
		{
			name: "images",
			start: func(app *AppModel) {
				app.imageModel, _ = app.imageModel.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
			},
		},
		{
			name: "volumes",
			start: func(app *AppModel) {
				app.volumeModel, _ = app.volumeModel.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
			},
		},
		{
			name: "networks",
			start: func(app *AppModel) {
				app.networkModel, _ = app.networkModel.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := NewApp(helperMockService())
			if app.isAnySearching() {
				t.Fatal("new app should not be searching")
			}

			tt.start(&app)
			if !app.isAnySearching() {
				t.Error("isAnySearching() = false after activating search")
			}
		})
	}
}

func TestAppModel_Update_SearchingKeys(t *testing.T) {
	app := NewApp(helperMockService())
	app.ready = true

	cancelCalled := false
	app.eventCancel = func() { cancelCalled = true }
	app.containerModel, _ = app.containerModel.Update(tea.KeyPressMsg{Code: '/', Text: "/"})

	newModel, _ := app.Update(tea.KeyPressMsg{Code: '2', Text: "2"})
	app = newModel.(AppModel)
	if app.activeTab != TabContainers {
		t.Errorf("activeTab while searching = %v, want TabContainers", app.activeTab)
	}
	if cancelCalled {
		t.Error("typing a tab key while searching should not cancel the app")
	}

	newModel, _ = app.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	app = newModel.(AppModel)
	if cancelCalled {
		t.Error("typing q while searching should not quit the app")
	}

	newModel, cmd := app.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	app = newModel.(AppModel)
	if !cancelCalled {
		t.Error("Ctrl+C while searching should cancel the event stream")
	}
	if cmd == nil {
		t.Fatal("Ctrl+C while searching should return a quit command")
	}
	quitMsg := cmd()
	if _, ok := quitMsg.(tea.QuitMsg); !ok {
		t.Errorf("Ctrl+C command result = %T, want tea.QuitMsg", quitMsg)
	}
}

func TestAppModel_View(t *testing.T) {
	mockSvc := helperMockService()
	app := NewApp(mockSvc)

	// Not ready View
	viewNotReady := app.View()
	if !strings.Contains(viewNotReady.Content, "Initializing") {
		t.Errorf("View when not ready should say Initializing, got %q", viewNotReady.Content)
	}

	// Ready View
	newModel, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = newModel.(AppModel)
	viewReady := app.View()
	renderStr := viewReady.Content

	if !strings.Contains(renderStr, "Containers") {
		t.Errorf("View should render tab header Containers, got %q", renderStr)
	}
}
