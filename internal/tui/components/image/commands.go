package image

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
	mobyimage "github.com/moby/moby/api/types/image"
	"github.com/xoaiPro235/docktop/internal/docker"
	"github.com/xoaiPro235/docktop/internal/tui/modal"
)

type ImageListMsg []docker.ImageInfo

type ImageDetailsMsg struct {
	ImageID string
	Details *mobyimage.InspectResponse
	Layers  []docker.ImageLayer
	Err     error
}

type ImageActionMsg struct {
	Action modal.Action
	Err    error
}

func (m Model) fetchImageDetailsCmd(imageID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		defer cancel()

		details, _ := m.dockerClient.InspectImage(ctx, imageID)
		layers, err := m.dockerClient.GetImageHistory(ctx, imageID)
		return ImageDetailsMsg{
			ImageID: imageID,
			Details: details,
			Layers:  layers,
			Err:     err,
		}
	}
}

func (m Model) removeImageCmd(targetID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		err := m.dockerClient.RemoveImage(ctx, targetID, false)
		return ImageActionMsg{Action: modal.ActionRemove, Err: err}
	}
}

func (m Model) pruneImagesCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		_, err := m.dockerClient.PruneImages(ctx)
		return ImageActionMsg{Action: modal.ActionPrune, Err: err}
	}
}
