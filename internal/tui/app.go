package tui

import (
	"context"
	"fmt"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/moby/moby/api/types/events"
	"github.com/xoaiPro235/docktop/internal/docker"
	"github.com/xoaiPro235/docktop/internal/tui/components/container"
	"github.com/xoaiPro235/docktop/internal/tui/components/image"
	"github.com/xoaiPro235/docktop/internal/tui/components/network"
	"github.com/xoaiPro235/docktop/internal/tui/components/volume"
	"github.com/xoaiPro235/docktop/internal/tui/keys"
	"github.com/xoaiPro235/docktop/internal/tui/modal"
	"github.com/xoaiPro235/docktop/internal/tui/theme"
)

type ContainerListMsg []docker.ContainerInfo
type ImageListMsg []docker.ImageInfo
type VolumeListMsg []docker.VolumeInfo
type NetworkListMsg []docker.NetworkInfo
type DockerEventMsg events.Message
type DockerEventStreamClosedMsg struct{}
type DockerReconnectTickMsg struct{}
type DockerManualReconnectResultMsg struct {
	Err error
}

const maxReconnectAttempts = 5

func (a AppModel) attemptReconnectCmd() tea.Cmd {
	return func() tea.Msg {
		err := a.dockerClient.Ping()
		return DockerManualReconnectResultMsg{Err: err}
	}
}
func waitForDockerEvent(eventCh <-chan events.Message, errCh <-chan error) tea.Cmd {
	return func() tea.Msg {
		if eventCh == nil && errCh == nil {
			return DockerEventStreamClosedMsg{}
		}
		select {
		case msg, ok := <-eventCh:
			if !ok {
				return DockerEventStreamClosedMsg{}
			}
			return DockerEventMsg(msg)
		case _, ok := <-errCh:
			// When Docker daemon disconnects or exits, moby client sends error to errCh and closes it.
			_ = ok
			return DockerEventStreamClosedMsg{}
		}
	}
}

func scheduleReconnectTick() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return DockerReconnectTickMsg{}
	})
}

type Tab int

const (
	TabContainers Tab = iota
	TabImages
	TabVolumes
	TabNetworks
)

type TabDef struct {
	Tab   Tab
	Title string
}

var Tabs = []TabDef{
	{Tab: TabContainers, Title: "1: Containers"},
	{Tab: TabImages, Title: "2: Images"},
	{Tab: TabVolumes, Title: "3: Volumes"},
	{Tab: TabNetworks, Title: "4: Networks"},
}

type AppModel struct {
	dockerClient      docker.Service
	containerModel    container.Model
	imageModel        image.Model
	volumeModel       volume.Model
	networkModel      network.Model
	errorModal        modal.ErrorModal
	activeTab         Tab
	eventCancel       context.CancelFunc
	eventChan         <-chan events.Message
	eventErrChan      <-chan error
	daemonConnected   bool
	reconnecting      bool
	reconnectAttempts int
	width             int
	height            int
	ready             bool
}

func NewApp(cli docker.Service) AppModel {
	ctx, cancel := context.WithCancel(context.Background())
	eventCh, errCh := cli.StreamEvents(ctx)

	return AppModel{
		dockerClient:      cli,
		containerModel:    container.New(cli),
		imageModel:        image.New(cli),
		volumeModel:       volume.New(cli),
		networkModel:      network.New(cli),
		errorModal:        modal.NewErrorModal(),
		activeTab:         TabContainers,
		eventCancel:       cancel,
		eventChan:         eventCh,
		eventErrChan:      errCh,
		daemonConnected:   true,
		reconnecting:      false,
		reconnectAttempts: 0,
	}
}

// fetchContainersCmd fetches fresh container list from Docker daemon
func (a AppModel) fetchContainersCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		list, err := a.dockerClient.ListContainers(ctx)
		if err != nil {
			return ContainerListMsg([]docker.ContainerInfo{})
		}
		return ContainerListMsg(list)
	}
}

// fetchImagesCmd fetches fresh image list from Docker daemon
func (a AppModel) fetchImagesCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		list, err := a.dockerClient.ListImages(ctx)
		if err != nil {
			return ImageListMsg([]docker.ImageInfo{})
		}
		return ImageListMsg(list)
	}
}

// fetchVolumesCmd fetches fresh volume list from Docker daemon
func (a AppModel) fetchVolumesCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		list, err := a.dockerClient.ListVolumes(ctx)
		if err != nil {
			return VolumeListMsg([]docker.VolumeInfo{})
		}
		return VolumeListMsg(list)
	}
}

// fetchNetworksCmd fetches fresh network list from Docker daemon
func (a AppModel) fetchNetworksCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		list, err := a.dockerClient.ListNetworks(ctx)
		if err != nil {
			return NetworkListMsg([]docker.NetworkInfo{})
		}
		return NetworkListMsg(list)
	}
}

func (a *AppModel) switchTab(tab Tab) tea.Cmd {
	if a.activeTab == tab {
		return a.fetchActiveTabCmd()
	}

	var cmds []tea.Cmd
	if tab == TabContainers {
		cmds = append(cmds, a.containerModel.Resume())
	} else if a.activeTab == TabContainers {
		a.containerModel.Pause()
	}

	a.activeTab = tab
	cmds = append(cmds, a.fetchActiveTabCmd())
	return tea.Batch(cmds...)
}

func (a AppModel) fetchActiveTabCmd() tea.Cmd {
	switch a.activeTab {
	case TabContainers:
		return a.fetchContainersCmd()
	case TabImages:
		return a.fetchImagesCmd()
	case TabVolumes:
		return a.fetchVolumesCmd()
	case TabNetworks:
		return a.fetchNetworksCmd()
	default:
		return nil
	}
}

func (a *AppModel) handleDaemonReconnect() []tea.Cmd {
	if a.daemonConnected {
		return nil
	}
	a.daemonConnected = true
	a.reconnecting = false
	a.reconnectAttempts = 0

	if a.eventCancel != nil {
		a.eventCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	newEventCh, newErrCh := a.dockerClient.StreamEvents(ctx)
	a.eventCancel = cancel
	a.eventChan = newEventCh
	a.eventErrChan = newErrCh

	return []tea.Cmd{
		waitForDockerEvent(a.eventChan, a.eventErrChan),
		a.fetchContainersCmd(),
		a.fetchImagesCmd(),
		a.fetchVolumesCmd(),
		a.fetchNetworksCmd(),
	}
}

func (a AppModel) Init() tea.Cmd {
	return tea.Batch(
		a.containerModel.Init(),
		a.imageModel.Init(),
		a.volumeModel.Init(),
		a.networkModel.Init(),
		a.fetchContainersCmd(),
		a.fetchImagesCmd(),
		a.fetchVolumesCmd(),
		a.fetchNetworksCmd(),
		waitForDockerEvent(a.eventChan, a.eventErrChan),
	)
}

func (a AppModel) isAnyModalActive() bool {
	return a.errorModal.Active ||
		a.containerModel.IsModalActive() ||
		a.imageModel.IsModalActive() ||
		a.volumeModel.IsModalActive() ||
		a.networkModel.IsModalActive()
}

func (a AppModel) isAnySearching() bool {
	return a.containerModel.IsSearching() ||
		a.imageModel.IsSearching() ||
		a.volumeModel.IsSearching() ||
		a.networkModel.IsSearching()
}

func (a AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if a.errorModal.Active {
		switch msg := msg.(type) {
		case tea.KeyPressMsg, tea.MouseClickMsg:
			var cmd tea.Cmd
			a.errorModal, cmd = a.errorModal.Update(msg)
			return a, cmd
		}
	}

	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if a.isAnySearching() {
			if key.Matches(msg, keys.AppKeys.ForceQuit) {
				if a.eventCancel != nil {
					a.eventCancel()
				}
				a.containerModel.Pause()
				return a, tea.Quit
			}
			break
		}

		switch {
		case key.Matches(msg, keys.AppKeys.Quit):
			if a.eventCancel != nil {
				a.eventCancel()
			}
			a.containerModel.Pause()
			return a, tea.Quit

		case key.Matches(msg, keys.AppKeys.Reconnect):
			if !a.isAnyModalActive() {
				a.reconnecting = true
				a.daemonConnected = false
				a.reconnectAttempts = 0
				return a, a.attemptReconnectCmd()
			}

		case key.Matches(msg, keys.AppKeys.TabContainers):
			if !a.isAnyModalActive() {
				return a, a.switchTab(TabContainers)
			}

		case key.Matches(msg, keys.AppKeys.TabImages):
			if !a.isAnyModalActive() {
				return a, a.switchTab(TabImages)
			}

		case key.Matches(msg, keys.AppKeys.TabVolumes):
			if !a.isAnyModalActive() {
				return a, a.switchTab(TabVolumes)
			}

		case key.Matches(msg, keys.AppKeys.TabNetworks):
			if !a.isAnyModalActive() {
				return a, a.switchTab(TabNetworks)
			}
		}

	case tea.MouseClickMsg:
		if msg.Y == 0 && !a.isAnyModalActive() {
			currX := 0
			for _, t := range Tabs {
				style := theme.Current.TabInactive
				if a.activeTab == t.Tab {
					style = theme.Current.TabActive
				}
				w := ansi.StringWidth(style.Render(t.Title))
				if msg.X >= currX && msg.X < currX+w {
					return a, a.switchTab(t.Tab)
				}
				currX += w
			}
		}

	case DockerEventMsg:
		// Clean event-driven dispatch: Only trigger refreshes for actual lifecycle changes
		switch msg.Type {
		case events.ContainerEventType:
			if docker.IsContainerLifecycleEvent(msg.Action) {
				cmds = append(cmds, a.fetchContainersCmd())
			}
		case events.ImageEventType:
			if docker.IsImageLifecycleEvent(msg.Action) {
				cmds = append(cmds, a.fetchImagesCmd())
			}
		case events.VolumeEventType:
			if docker.IsVolumeLifecycleEvent(msg.Action) {
				cmds = append(cmds, a.fetchVolumesCmd())
			}
		case events.NetworkEventType:
			if docker.IsNetworkLifecycleEvent(msg.Action) {
				cmds = append(cmds, a.fetchNetworksCmd())
			}
		}
		cmds = append(cmds, waitForDockerEvent(a.eventChan, a.eventErrChan))

	case DockerEventStreamClosedMsg:
		if !a.reconnecting {
			a.daemonConnected = false
			a.reconnecting = true
			a.reconnectAttempts = 1
			cmds = append(cmds, scheduleReconnectTick())
		}
		return a, tea.Batch(cmds...)

	case DockerReconnectTickMsg:
		if !a.reconnecting {
			return a, nil
		}
		if err := a.dockerClient.Ping(); err != nil {
			a.reconnectAttempts++
			if a.reconnectAttempts > maxReconnectAttempts {
				a.reconnecting = false
				a.daemonConnected = false
				a.reconnectAttempts = 0
				return a, nil
			}
			return a, scheduleReconnectTick()
		}

		// Reconnect successful! Daemon is back online.
		cmds = append(cmds, a.handleDaemonReconnect()...)
		return a, tea.Batch(cmds...)

	case DockerManualReconnectResultMsg:
		if msg.Err != nil {
			a.reconnecting = false
			a.daemonConnected = false
			a.reconnectAttempts = 0
			rawHost := a.dockerClient.Host()
			a.errorModal.Show(
				"Docker Connection Failed",
				fmt.Sprintf("%s\n\nError: %v", rawHost, msg.Err),
			)
			return a, nil
		}

		// Manual reconnect successful! Daemon is back online.
		cmds = append(cmds, a.handleDaemonReconnect()...)
		return a, tea.Batch(cmds...)

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.ready = true
		// Subtract 3 lines for header tab bar and footer help
		a.containerModel.SetSize(msg.Width, msg.Height-3)
		a.imageModel.SetSize(msg.Width, msg.Height-3)
		a.volumeModel.SetSize(msg.Width, msg.Height-3)
		a.networkModel.SetSize(msg.Width, msg.Height-3)
		return a, nil

	case ContainerListMsg:
		if !a.daemonConnected {
			cmds = append(cmds, a.handleDaemonReconnect()...)
		}
		if cmd := a.containerModel.SetContainers(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}

	case ImageListMsg:
		if !a.daemonConnected {
			cmds = append(cmds, a.handleDaemonReconnect()...)
		}
		if cmd := a.imageModel.SetImages(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}

	case VolumeListMsg:
		if !a.daemonConnected {
			cmds = append(cmds, a.handleDaemonReconnect()...)
		}
		if cmd := a.volumeModel.SetVolumes(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}

	case NetworkListMsg:
		if !a.daemonConnected {
			cmds = append(cmds, a.handleDaemonReconnect()...)
		}
		if cmd := a.networkModel.SetNetworks(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}

	case modal.ShowErrorMsg:
		a.errorModal.Show(msg.Title, msg.Message)
		return a, nil

	case container.ContainerActionMsg:
		if msg.Err != nil {
			a.errorModal.Show(fmt.Sprintf("Container %s Failed", msg.Action.Title()), msg.Err.Error())
		} else if !a.daemonConnected {
			cmds = append(cmds, a.handleDaemonReconnect()...)
		} else {
			cmds = append(cmds, a.fetchContainersCmd())
		}
		var cmd tea.Cmd
		a.containerModel, cmd = a.containerModel.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return a, tea.Batch(cmds...)

	case image.ImageActionMsg:
		if msg.Err != nil {
			a.errorModal.Show(fmt.Sprintf("Image %s Failed", msg.Action.Title()), msg.Err.Error())
		} else if !a.daemonConnected {
			cmds = append(cmds, a.handleDaemonReconnect()...)
		} else {
			cmds = append(cmds, a.fetchImagesCmd())
		}
		var cmd tea.Cmd
		a.imageModel, cmd = a.imageModel.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return a, tea.Batch(cmds...)

	case volume.VolumeActionMsg:
		if msg.Err != nil {
			a.errorModal.Show(fmt.Sprintf("Volume %s Failed", msg.Action.Title()), msg.Err.Error())
		} else if !a.daemonConnected {
			cmds = append(cmds, a.handleDaemonReconnect()...)
		} else {
			cmds = append(cmds, a.fetchVolumesCmd())
		}
		var cmd tea.Cmd
		a.volumeModel, cmd = a.volumeModel.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return a, tea.Batch(cmds...)

	case network.NetworkActionMsg:
		if msg.Err != nil {
			a.errorModal.Show(fmt.Sprintf("Network %s Failed", msg.Action.Title()), msg.Err.Error())
		} else if !a.daemonConnected {
			cmds = append(cmds, a.handleDaemonReconnect()...)
		} else {
			cmds = append(cmds, a.fetchNetworksCmd())
		}
		var cmd tea.Cmd
		a.networkModel, cmd = a.networkModel.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return a, tea.Batch(cmds...)

	case container.ExecFinishedMsg:
		if msg.Err != nil {
			a.errorModal.Show("Container Exec Failed", msg.Err.Error())
		}
		var cmd tea.Cmd
		a.containerModel, cmd = a.containerModel.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return a, tea.Batch(cmds...)

	case spinner.TickMsg:
		var routed bool
		if a.containerModel.IsActionRunning() {
			var cmd tea.Cmd
			a.containerModel, cmd = a.containerModel.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			routed = true
		}
		if a.imageModel.IsActionRunning() {
			var cmd tea.Cmd
			a.imageModel, cmd = a.imageModel.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			routed = true
		}
		if a.volumeModel.IsActionRunning() {
			var cmd tea.Cmd
			a.volumeModel, cmd = a.volumeModel.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			routed = true
		}
		if a.networkModel.IsActionRunning() {
			var cmd tea.Cmd
			a.networkModel, cmd = a.networkModel.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			routed = true
		}
		if routed {
			return a, tea.Batch(cmds...)
		}
		return a, nil

	case container.PollTopMsg,
		container.StatsResultMsg,
		container.StatsStreamClosedMsg,
		container.TopLoadedMsg,
		container.InspectLoadedMsg,
		container.LogBatchMsg,
		container.LogStreamClosedMsg:
		if a.activeTab != TabContainers {
			return a, nil
		}
		var cmd tea.Cmd
		a.containerModel, cmd = a.containerModel.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return a, tea.Batch(cmds...)

	case image.ImageDetailsMsg:
		var cmd tea.Cmd
		a.imageModel, cmd = a.imageModel.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return a, tea.Batch(cmds...)
	}

	// Forward remaining UI / user interaction messages to currently active tab
	switch a.activeTab {
	case TabContainers:
		var cmd tea.Cmd
		a.containerModel, cmd = a.containerModel.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case TabImages:
		var cmd tea.Cmd
		a.imageModel, cmd = a.imageModel.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case TabVolumes:
		var cmd tea.Cmd
		a.volumeModel, cmd = a.volumeModel.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case TabNetworks:
		var cmd tea.Cmd
		a.networkModel, cmd = a.networkModel.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return a, tea.Batch(cmds...)
}

func (a AppModel) FooterBindings() []key.Binding {
	if a.errorModal.Active {
		return a.errorModal.FooterBindings()
	}

	var bindings []key.Binding
	switch a.activeTab {
	case TabContainers:
		bindings = a.containerModel.FooterBindings()
	case TabImages:
		bindings = a.imageModel.FooterBindings()
	case TabVolumes:
		bindings = a.volumeModel.FooterBindings()
	case TabNetworks:
		bindings = a.networkModel.FooterBindings()
	}

	if !a.daemonConnected && !a.reconnecting {
		bindings = append([]key.Binding{keys.AppKeys.Reconnect}, bindings...)
	}

	return bindings
}

func (a AppModel) View() tea.View {
	if !a.ready {
		v := tea.NewView("Initializing docktop...")
		v.AltScreen = true
		v.MouseMode = tea.MouseModeCellMotion
		return v
	}

	// 1. Header Tabs
	var tabViews []string
	for _, t := range Tabs {
		style := theme.Current.TabInactive
		if a.activeTab == t.Tab {
			style = theme.Current.TabActive
		}
		tabViews = append(tabViews, style.Render(t.Title))
	}
	headerTabs := lipgloss.JoinHorizontal(lipgloss.Top, tabViews...)

	var statusBadge string
	if a.reconnecting {
		if a.reconnectAttempts > 0 {
			statusBadge = theme.Current.StatusPaused.Render(fmt.Sprintf("○ RECONNECTING (%d/%d)...", a.reconnectAttempts, maxReconnectAttempts))
		} else {
			statusBadge = theme.Current.StatusPaused.Render("○ RECONNECTING...")
		}
	} else if a.daemonConnected {
		statusBadge = theme.Current.StatusRunning.Render("● CONNECTED")
	} else {
		statusBadge = theme.Current.StatusStopped.Render("○ DISCONNECTED")
	}

	rawHost := a.dockerClient.Host()
	hostLabel := theme.Current.Key.Render("Host: ")
	hostValue := theme.Current.Value.Render(rawHost)
	fullRight := hostLabel + hostValue + "  " + statusBadge

	leftWidth := ansi.StringWidth(headerTabs)
	rightWidth := a.width - leftWidth

	var rightView string
	if rightWidth > 0 {
		var rightContent string
		if rightWidth >= ansi.StringWidth(fullRight)+2 {
			rightContent = fullRight
		} else if rightWidth >= ansi.StringWidth(statusBadge)+2 {
			rightContent = statusBadge
		}
		rightView = lipgloss.NewStyle().
			Width(rightWidth).
			Align(lipgloss.Right).
			PaddingRight(1).
			Render(rightContent)
	}

	header := lipgloss.JoinHorizontal(lipgloss.Top, headerTabs, rightView)

	// 2. Tab content
	var body string
	switch a.activeTab {
	case TabContainers:
		body = a.containerModel.View()
	case TabImages:
		body = a.imageModel.View()
	case TabVolumes:
		body = a.volumeModel.View()
	case TabNetworks:
		body = a.networkModel.View()
	}

	// 3. Dynamic Footer Bar based on active tab & subpane
	footerHelp := keys.RenderFooterHelp(a.FooterBindings()...)
	footer := theme.Current.Footer.Width(a.width).Render(footerHelp)

	out := lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
	if a.errorModal.Active {
		out = a.errorModal.Overlay(out, a.width, a.height)
	}

	v := tea.NewView(out)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}
