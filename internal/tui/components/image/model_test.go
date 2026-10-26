package image

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"
	mobyimage "github.com/moby/moby/api/types/image"
	"github.com/xoaiPro235/docktop/internal/docker"
	"github.com/xoaiPro235/docktop/internal/docker/mocks"
	"github.com/xoaiPro235/docktop/internal/tui/modal"
	"github.com/xoaiPro235/docktop/internal/tui/theme"
)

func init() {
	theme.SetThemeByName("tokyo-night")
}

func TestNew(t *testing.T) {
	mockCli := &mocks.MockImageService{}
	m := New(mockCli)

	if m.dockerClient == nil {
		t.Error("Expected dockerClient to be assigned")
	}
	if m.focusZone != FocusImagesList {
		t.Errorf("focusZone = %v, want FocusImagesList", m.focusZone)
	}
	if m.IsModalActive() {
		t.Error("New model should have inactive modal")
	}
	if m.IsActionRunning() {
		t.Error("New model should not have running action")
	}
}

func TestModel_Init(t *testing.T) {
	mockCli := &mocks.MockImageService{}
	m := New(mockCli)
	if cmd := m.Init(); cmd != nil {
		t.Errorf("Init() = %v, want nil", cmd)
	}
}

func TestModel_SetSize(t *testing.T) {
	mockCli := &mocks.MockImageService{}
	m := New(mockCli)
	m.SetSize(120, 40)

	if m.width != 120 || m.height != 40 {
		t.Errorf("width, height = (%d, %d), want (120, 40)", m.width, m.height)
	}
	if !m.viewportReady {
		t.Error("viewportReady should be true after SetSize")
	}
}

func TestModel_SetImages(t *testing.T) {
	mockCli := &mocks.MockImageService{
		InspectImageFunc: func(ctx context.Context, id string) (*mobyimage.InspectResponse, error) {
			return &mobyimage.InspectResponse{ID: id}, nil
		},
		GetImageHistoryFunc: func(ctx context.Context, id string) ([]docker.ImageLayer, error) {
			return []docker.ImageLayer{{Index: 0, Command: "CMD"}}, nil
		},
	}
	m := New(mockCli)
	m.SetSize(100, 30)

	imgs := []docker.ImageInfo{
		{ID: "img-1", Tag: "alpine:3.18"},
		{ID: "img-2", Tag: "postgres:16"},
	}

	// 1. Should return fetch cmd because details/layers not loaded
	cmd := m.SetImages(imgs)
	if cmd == nil {
		t.Error("Expected fetchImageDetailsCmd when layers are empty")
	}

	// 2. Cursor index tracking
	m.cursorIndex = 1
	newImgs := []docker.ImageInfo{
		{ID: "img-2", Tag: "postgres:16", Details: &mobyimage.InspectResponse{ID: "img-2"}, Layers: []docker.ImageLayer{{Index: 0}}},
		{ID: "img-1", Tag: "alpine:3.18"},
	}
	cmd2 := m.SetImages(newImgs)
	if m.SelectedImage() == nil || m.SelectedImage().ID != "img-2" {
		t.Errorf("SelectedImage after reorder = %v, want img-2", m.SelectedImage())
	}
	if cmd2 != nil {
		t.Error("Expected nil cmd when selected image has details and layers cached")
	}
}

func TestModel_FilterImages(t *testing.T) {
	m := New(&mocks.MockImageService{})
	m.SetImages([]docker.ImageInfo{
		{ID: "img-1", Tag: "nginx:latest", Details: &mobyimage.InspectResponse{ID: "img-1"}, Layers: []docker.ImageLayer{{Index: 0}}},
		{ID: "img-2", Tag: "redis:7", Details: &mobyimage.InspectResponse{ID: "img-2"}, Layers: []docker.ImageLayer{{Index: 0}}},
	})

	sendKey := func(msg tea.Msg) {
		m, _ = m.Update(msg)
	}
	sendKey(tea.KeyPressMsg{Code: '/', Text: "/"})
	for _, r := range "NGINX" {
		sendKey(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	sendKey(tea.KeyPressMsg{Code: tea.KeyEnter})

	if m.IsSearching() {
		t.Error("search input should be inactive after submitting the filter")
	}
	if got := m.SelectedImage(); got == nil || got.ID != "img-1" {
		t.Errorf("SelectedImage() after filtering = %v, want img-1", got)
	}
	if len(m.filteredIndices) != 1 {
		t.Errorf("len(filteredIndices) after filtering = %d, want 1", len(m.filteredIndices))
	}

	sendKey(tea.KeyPressMsg{Code: tea.KeyEsc})
	if len(m.filteredIndices) != 2 {
		t.Errorf("len(filteredIndices) after clearing filter = %d, want 2", len(m.filteredIndices))
	}
}

func TestModel_IsModalActive(t *testing.T) {
	mockCli := &mocks.MockImageService{}
	m := New(mockCli)
	if m.IsModalActive() {
		t.Error("IsModalActive should be false initially")
	}

	m.confirmModal.ShowImageAction(modal.ActionRemove, "img-1", "alpine:latest")
	if !m.IsModalActive() {
		t.Error("IsModalActive should be true when modal is shown")
	}
}

func TestModel_IsActionRunning(t *testing.T) {
	mockCli := &mocks.MockImageService{}
	m := New(mockCli)
	if m.IsActionRunning() {
		t.Error("IsActionRunning should be false initially")
	}

	m.action.Start(modal.ActionRemove, "img-1", "alpine:latest", "Removing image")
	if !m.IsActionRunning() {
		t.Error("IsActionRunning should be true after action.Start")
	}
}

func TestModel_SelectedImage(t *testing.T) {
	mockCli := &mocks.MockImageService{}
	m := New(mockCli)

	if got := m.SelectedImage(); got != nil {
		t.Errorf("SelectedImage() on empty list = %v, want nil", got)
	}

	m.images = []docker.ImageInfo{
		{ID: "img-alpha", Tag: "golang:1.24"},
		{ID: "img-beta", Tag: "redis:7"},
	}
	m.applyFilter()
	m.cursorIndex = 1

	if got := m.SelectedImage(); got == nil || got.ID != "img-beta" {
		t.Errorf("SelectedImage() = %v, want img-beta", got)
	}
}

func TestModel_updateViewportContent(t *testing.T) {
	mockCli := &mocks.MockImageService{}
	m := New(mockCli)

	// Before viewport is ready
	m.updateViewportContent()

	// After viewport is ready
	m.SetSize(100, 30)
	m.updateViewportContent()

	// With populated image
	m.images = []docker.ImageInfo{
		{
			ID:       "sha256:1234567890abcdef",
			Tag:      "node:20-alpine",
			Size:     150 * 1024 * 1024,
			RepoTags: []string{"node:20-alpine"},
			Layers: []docker.ImageLayer{
				{Index: 0, Size: 5 * 1024 * 1024, Command: "RUN apk add --no-cache curl"},
			},
		},
	}
	m.applyFilter()
	m.updateViewportContent()
}

func TestModel_syncDetails(t *testing.T) {
	mockCli := &mocks.MockImageService{}
	m := New(mockCli)
	m.SetSize(100, 30)
	m.images = []docker.ImageInfo{
		{ID: "img-1", Tag: "nginx:latest"},
	}
	m.applyFilter()

	cmd := m.syncDetails()
	if cmd == nil {
		t.Error("syncDetails should return fetch cmd when details are missing")
	}
}
