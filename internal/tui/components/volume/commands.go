package volume

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/xoaiPro235/docktop/internal/docker"
	"github.com/xoaiPro235/docktop/internal/tui/modal"
)

type VolumeListMsg []docker.VolumeInfo

type VolumeActionMsg struct {
	Action modal.Action
	Err    error
}

func (m Model) removeVolumeCmd(name string, force bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := m.dockerClient.RemoveVolume(ctx, name, force)
		return VolumeActionMsg{Action: modal.ActionRemove, Err: err}
	}
}

func (m Model) pruneVolumesCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_, err := m.dockerClient.PruneVolumes(ctx)
		return VolumeActionMsg{Action: modal.ActionPrune, Err: err}
	}
}
