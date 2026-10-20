package container

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/moby/moby/api/types/container"
	"github.com/xoaiPro235/docktop/internal/docker"
	"github.com/xoaiPro235/docktop/internal/tui/modal"
)

type PollTopMsg struct {
	ContainerID string
}

type LogBatchMsg struct {
	ContainerID string
	Lines       []string
}

type LogStreamClosedMsg struct {
	ContainerID string
}

type StatsResultMsg struct {
	ContainerID string
	Metrics     docker.ContainerMetrics
}

type StatsStreamClosedMsg struct {
	ContainerID string
}

type TopLoadedMsg struct {
	ContainerID string
	Data        *container.TopResponse
	Err         error
}

type ExecFinishedMsg struct {
	Err error
}

type ContainerActionMsg struct {
	ContainerID string
	Action      modal.Action
	Err         error
}

// ExecuteActionMsg triggers an asynchronous container lifecycle action.
type ExecuteActionMsg struct {
	Action     modal.Action
	TargetID   string
	TargetName string
}

// pollTopAfter schedules a PollTopMsg after duration d
func (m Model) pollTopAfter(d time.Duration, containerID string) tea.Cmd {
	return tea.Tick(d, func(t time.Time) tea.Msg {
		return PollTopMsg{ContainerID: containerID}
	})
}

func (m Model) fetchInspectCmd(containerID string) tea.Cmd {
	return func() tea.Msg {
		if containerID == "" {
			return nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		data, err := m.dockerClient.InspectContainer(ctx, containerID)
		return InspectLoadedMsg{
			ContainerID: containerID,
			Data:        data,
			Err:         err,
		}
	}
}

func (m Model) fetchTopCmd(containerID string) tea.Cmd {
	return func() tea.Msg {
		if containerID == "" {
			return nil
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		top, err := m.dockerClient.ContainerTop(ctx, containerID)
		return TopLoadedMsg{
			ContainerID: containerID,
			Data:        top,
			Err:         err,
		}
	}
}

func (m Model) startContainerCmd(id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := m.dockerClient.StartContainer(ctx, id)
		return ContainerActionMsg{ContainerID: id, Action: modal.ActionStart, Err: err}
	}
}

func (m Model) stopContainerCmd(id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		err := m.dockerClient.StopContainer(ctx, id)
		return ContainerActionMsg{ContainerID: id, Action: modal.ActionStop, Err: err}
	}
}

func (m Model) restartContainerCmd(id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		err := m.dockerClient.RestartContainer(ctx, id)
		return ContainerActionMsg{ContainerID: id, Action: modal.ActionRestart, Err: err}
	}
}

func (m Model) pauseContainerCmd(id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := m.dockerClient.PauseContainer(ctx, id)
		return ContainerActionMsg{ContainerID: id, Action: modal.ActionPause, Err: err}
	}
}

func (m Model) unpauseContainerCmd(id string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		err := m.dockerClient.UnpauseContainer(ctx, id)
		return ContainerActionMsg{ContainerID: id, Action: modal.ActionUnpause, Err: err}
	}
}

func (m Model) removeContainerCmd(id string, force bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		err := m.dockerClient.RemoveContainer(ctx, id, docker.RemoveContainerOpts{
			Force: force,
		})
		action := modal.ActionRemove
		if force {
			action = modal.ActionRemoveForce
		}
		return ContainerActionMsg{ContainerID: id, Action: action, Err: err}
	}
}

func (m Model) pruneContainersCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_, err := m.dockerClient.PruneContainers(ctx)
		return ContainerActionMsg{ContainerID: "", Action: modal.ActionPrune, Err: err}
	}
}

func (m *Model) startLogStream(containerID string) tea.Cmd {
	m.stopLogStream()
	m.logsPane.Reset()
	if containerID == "" {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.logCancel = cancel

	outCh := m.dockerClient.StreamContainerLogs(ctx, containerID, "100")
	m.logChan = outCh

	return WaitForNextLog(outCh, containerID)
}

func (m *Model) startComposeLogStream(node *TreeNode) tea.Cmd {
	m.stopLogStream()
	m.logsPane.Reset()
	if node == nil {
		return nil
	}

	projectName, serviceName, workDir, configFile := resolveComposeProject(node)
	if projectName == "" {
		m.logsPane.AppendLine("Compose project name not found.")
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.logCancel = cancel

	outCh := m.dockerClient.StreamComposeLogs(ctx, docker.ComposeLogOpts{
		WorkingDir:  workDir,
		ConfigFile:  configFile,
		ProjectName: projectName,
		Service:     serviceName,
		Tail:        "50",
	})
	m.logChan = outCh

	return WaitForNextLog(outCh, node.Key)
}

func WaitForNextLog(subCh <-chan string, containerID string) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-subCh
		if !ok {
			return LogStreamClosedMsg{ContainerID: containerID}
		}
		lines := []string{line}
		const maxBatch = 200
		for len(lines) < maxBatch {
			select {
			case l, ok := <-subCh:
				if !ok {
					return LogBatchMsg{
						ContainerID: containerID,
						Lines:       lines,
					}
				}
				lines = append(lines, l)
			default:
				return LogBatchMsg{
					ContainerID: containerID,
					Lines:       lines,
				}
			}
		}
		return LogBatchMsg{
			ContainerID: containerID,
			Lines:       lines,
		}
	}
}

func (m *Model) startStatsStream(containerID string) tea.Cmd {
	m.stopStatsStream()
	// NOTE: statsPane.Reset() is intentionally NOT called here.
	// History is preserved when resuming the tab (Pause → Resume).
	// Reset only happens in syncActivePane() when the selected container changes.
	if containerID == "" {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.statsCancel = cancel

	outCh := m.dockerClient.StreamContainerStats(ctx, containerID)
	m.statsChan = outCh

	return WaitForNextStat(outCh, containerID)
}

func WaitForNextStat(sub <-chan docker.ContainerMetrics, containerID string) tea.Cmd {
	return func() tea.Msg {
		metrics, ok := <-sub
		if !ok {
			// Stream closed (Docker error / daemon restart) → notify model to clear statsChan
			return StatsStreamClosedMsg{ContainerID: containerID}
		}
		return StatsResultMsg{ContainerID: containerID, Metrics: metrics}
	}
}
