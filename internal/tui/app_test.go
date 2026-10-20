package ui

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/moby/moby/api/types/events"
	"github.com/xoaiPro235/docktop/internal/docker"
	"github.com/xoaiPro235/docktop/internal/docker/mocks"
	"github.com/xoaiPro235/docktop/internal/tui/theme"
)

func init() {
	theme.SetThemeByName("tokyo-night")
}

func helperMockService() *mocks.MockService {
	eventCh := make(chan events.Message)
	return &mocks.MockService{
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
	// 1. Nil channel returns nil
	cmdNil := waitForDockerEvent(nil)
	if msg := cmdNil(); msg != nil {
		t.Errorf("waitForDockerEvent(nil)() = %v, want nil", msg)
	}

	// 2. Active channel with event
	ch := make(chan events.Message, 1)
	ch <- events.Message{Action: "start", Type: "container"}
	cmd := waitForDockerEvent(ch)
	msg := cmd()
	dockerMsg, ok := msg.(DockerEventMsg)
	if !ok || dockerMsg.Action != "start" {
		t.Errorf("Expected DockerEventMsg with action start, got %v", msg)
	}

	// 3. Closed channel returns nil
	close(ch)
	cmdClosed := waitForDockerEvent(ch)
	if msg := cmdClosed(); msg != nil {
		t.Errorf("waitForDockerEvent(closed)() = %v, want nil", msg)
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
	if app.isAnyModalActive() {
		t.Error("No modal should be active initially")
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
