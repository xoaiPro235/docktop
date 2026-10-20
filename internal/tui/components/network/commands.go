package network

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/xoaiPro235/docktop/internal/docker"
	"github.com/xoaiPro235/docktop/internal/tui/modal"
)

type NetworkListMsg []docker.NetworkInfo

type NetworkActionMsg struct {
	Action modal.Action
	Err    error
}

func (m Model) removeNetworkCmd(id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := m.dockerClient.RemoveNetwork(ctx, id)
		return NetworkActionMsg{Action: modal.ActionDelete, Err: err}
	}
}

func (m Model) pruneNetworksCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		_, err := m.dockerClient.PruneNetworks(ctx)
		return NetworkActionMsg{Action: modal.ActionPrune, Err: err}
	}
}
