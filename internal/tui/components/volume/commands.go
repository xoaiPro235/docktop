package volume

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/xoaiPro235/docktop/internal/docker"
)

type VolumeListMsg []docker.VolumeInfo

type VolumeResultMsg struct {
	TargetID string
	Err      error
}

type RunCmdMsg struct {
	TargetID string
	Cmd      tea.Cmd
}

func (m Model) removeVolumeCmd(name string, force bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := m.dockerClient.RemoveVolume(ctx, name, force)
		return VolumeResultMsg{TargetID: name, Err: err}
	}
}

func (m Model) pruneVolumesCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_, err := m.dockerClient.PruneVolumes(ctx)
		return VolumeResultMsg{TargetID: ActionTargetAll, Err: err}
	}
}
