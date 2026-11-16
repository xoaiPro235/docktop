package network

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/xoaiPro235/docktop/internal/docker"
)

type NetworkListMsg []docker.NetworkInfo

type NetworkResultMsg struct {
	TargetID string
	Err      error
}

type RunCmdMsg struct {
	TargetID string
	Cmd      tea.Cmd
}

func (m Model) removeNetworkCmd(id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := m.dockerClient.RemoveNetwork(ctx, id)
		return NetworkResultMsg{TargetID: id, Err: err}
	}
}

func (m Model) pruneNetworksCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_, err := m.dockerClient.PruneNetworks(ctx)
		return NetworkResultMsg{TargetID: ActionTargetAll, Err: err}
	}
}
