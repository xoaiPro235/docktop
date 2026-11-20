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
	"github.com/xoaiPro235/docktop/internal/config"
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

func (a AppModel) manualReconnectCmd() tea.Cmd {
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
	cfg               *config.Config
	containerModel    container.Model
	imageModel        image.Model
	volumeModel       volume.Model
	networkModel      network.Model
	confirmModal      modal.ConfirmModal
	errorModal        modal.ErrorModal
	actionMenuModal   modal.ActionMenuModal
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
	return NewAppWithConfig(cli, config.Default())
}

func NewAppWithConfig(cli docker.Service, cfg *config.Config) AppModel {
	if cfg == nil {
		cfg = config.Default()
	}
	ctx, cancel := context.WithCancel(context.Background())
	eventCh, errCh := cli.StreamEvents(ctx)

	return AppModel{
		dockerClient:      cli,
		cfg:               cfg,
		containerModel:    container.New(cli),
		imageModel:        image.New(cli),
		volumeModel:       volume.New(cli),
		networkModel:      network.New(cli),
		confirmModal:      modal.NewConfirmModal(),
		errorModal:        modal.NewErrorModal(),
		actionMenuModal:   modal.NewActionMenuModal(),
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

func (a AppModel) fetchTabCmd(tab Tab) tea.Cmd {
	switch tab {
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

func (a AppModel) fetchActiveTabCmd() tea.Cmd {
	return a.fetchTabCmd(a.activeTab)
}

// handleActionSuccess restores daemon connectivity if needed, and dispatches refreshes for the currently active tab.
func (a *AppModel) handleActionSuccess(resourceTab Tab) []tea.Cmd {
	if !a.daemonConnected {
		return a.handleDaemonReconnected()
	}
	// Container changes affect all tabs (mounts, networks, image usage).
	// Resource-specific changes only affect their own tab.
	// Inactive tabs are lazily refreshed upon switchTab.
	if resourceTab == TabContainers || a.activeTab == resourceTab {
		return []tea.Cmd{a.fetchActiveTabCmd()}
	}
	return nil
}

func (a *AppModel) handleDaemonReconnected() []tea.Cmd {
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
	return a.errorModal.Active || a.confirmModal.Active || a.actionMenuModal.Active
}

func (a AppModel) isAnySearching() bool {
	return a.containerModel.IsSearching() ||
		a.imageModel.IsSearching() ||
		a.volumeModel.IsSearching() ||
		a.networkModel.IsSearching()
}

func (a AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if a.confirmModal.Active {
		switch msg := msg.(type) {
		case tea.KeyPressMsg:
			var cmd tea.Cmd
			a.confirmModal, cmd = a.confirmModal.Update(msg)
			return a, cmd
		case tea.MouseClickMsg, tea.MouseWheelMsg:
			return a, nil
		}
	}

	if a.errorModal.Active {
		switch msg := msg.(type) {
		case tea.KeyPressMsg, tea.MouseClickMsg:
			var cmd tea.Cmd
			a.errorModal, cmd = a.errorModal.Update(msg)
			return a, cmd
		case tea.MouseWheelMsg:
			return a, nil
		}
	}

	if a.actionMenuModal.Active {
		switch msg := msg.(type) {
		case tea.KeyPressMsg:
			var cmd tea.Cmd
			a.actionMenuModal, cmd = a.actionMenuModal.Update(msg)
			return a, cmd
		case tea.MouseWheelMsg:
			var cmd tea.Cmd
			a.actionMenuModal, cmd = a.actionMenuModal.Update(msg)
			return a, cmd
		case tea.MouseClickMsg:
			return a, nil
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
				return a, a.manualReconnectCmd()
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

		case key.Matches(msg, keys.AppKeys.Menu):
			if !a.isAnyModalActive() && !a.isAnySearching() {
				cmd := a.openActionMenu()
				return a, cmd
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
				cmds = append(cmds, a.handleActionSuccess(TabContainers)...)
			}
		case events.ImageEventType:
			if docker.IsImageLifecycleEvent(msg.Action) {
				cmds = append(cmds, a.handleActionSuccess(TabImages)...)
			}
		case events.VolumeEventType:
			if docker.IsVolumeLifecycleEvent(msg.Action) {
				cmds = append(cmds, a.handleActionSuccess(TabVolumes)...)
			}
		case events.NetworkEventType:
			if docker.IsNetworkLifecycleEvent(msg.Action) {
				cmds = append(cmds, a.handleActionSuccess(TabNetworks)...)
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
		cmds = append(cmds, a.handleDaemonReconnected()...)
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
		cmds = append(cmds, a.handleDaemonReconnected()...)
		return a, tea.Batch(cmds...)

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.ready = true
		a.actionMenuModal.SetHeight(msg.Height)
		// Subtract 3 lines for header tab bar and footer help
		a.containerModel.SetSize(msg.Width, msg.Height-3)
		a.imageModel.SetSize(msg.Width, msg.Height-3)
		a.volumeModel.SetSize(msg.Width, msg.Height-3)
		a.networkModel.SetSize(msg.Width, msg.Height-3)
		return a, nil

	case ContainerListMsg:
		if !a.daemonConnected {
			cmds = append(cmds, a.handleDaemonReconnected()...)
		}
		if a.actionMenuModal.Active && a.activeTab == TabContainers && a.actionMenuModal.TargetID != "" {
			targetID := a.actionMenuModal.TargetID
			targetStillExists := false
			for _, c := range msg {
				if c.ID == targetID {
					targetStillExists = true
					break
				}
				if c.ProjectName != "" {
					if targetID == c.ProjectName || targetID == "proj:"+c.ProjectName {
						targetStillExists = true
						break
					}
					svcName := c.Service
					if svcName == "" {
						svcName = "other"
					}
					if targetID == fmt.Sprintf("svc:%s/%s", c.ProjectName, svcName) {
						targetStillExists = true
						break
					}
				}
			}
			if !targetStillExists {
				a.actionMenuModal.Hide()
			}
		}
		if cmd := a.containerModel.SetContainers(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}

	case ImageListMsg:
		if !a.daemonConnected {
			cmds = append(cmds, a.handleDaemonReconnected()...)
		}
		if a.actionMenuModal.Active && a.activeTab == TabImages && a.actionMenuModal.TargetID != "" {
			targetStillExists := false
			for _, img := range msg {
				if img.ID == a.actionMenuModal.TargetID {
					targetStillExists = true
					break
				}
			}
			if !targetStillExists {
				a.actionMenuModal.Hide()
			}
		}
		if cmd := a.imageModel.SetImages(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}

	case VolumeListMsg:
		if !a.daemonConnected {
			cmds = append(cmds, a.handleDaemonReconnected()...)
		}
		if a.actionMenuModal.Active && a.activeTab == TabVolumes && a.actionMenuModal.TargetID != "" {
			targetStillExists := false
			for _, vol := range msg {
				if vol.Name == a.actionMenuModal.TargetID {
					targetStillExists = true
					break
				}
			}
			if !targetStillExists {
				a.actionMenuModal.Hide()
			}
		}
		if cmd := a.volumeModel.SetVolumes(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}

	case NetworkListMsg:
		if !a.daemonConnected {
			cmds = append(cmds, a.handleDaemonReconnected()...)
		}
		if a.actionMenuModal.Active && a.activeTab == TabNetworks && a.actionMenuModal.TargetID != "" {
			targetStillExists := false
			for _, net := range msg {
				if net.ID == a.actionMenuModal.TargetID {
					targetStillExists = true
					break
				}
			}
			if !targetStillExists {
				a.actionMenuModal.Hide()
			}
		}
		if cmd := a.networkModel.SetNetworks(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}

	case modal.ShowConfirmMsg:
		a.confirmModal.Show(msg.Question, msg.Command, msg.Style, msg.OnConfirm)
		return a, nil

	case modal.ShowErrorMsg:
		a.errorModal.Show(msg.Title, msg.Message)
		return a, nil

	case CustomCommandFinishedMsg:
		if msg.Err != nil {
			a.errorModal.Show("Custom Command Error", msg.Err.Error())
		}
		return a, nil

	case container.RunCmdMsg:
		var cmd tea.Cmd
		a.containerModel, cmd = a.containerModel.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return a, tea.Batch(cmds...)

	case image.RunCmdMsg:
		var cmd tea.Cmd
		a.imageModel, cmd = a.imageModel.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return a, tea.Batch(cmds...)

	case volume.RunCmdMsg:
		var cmd tea.Cmd
		a.volumeModel, cmd = a.volumeModel.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return a, tea.Batch(cmds...)

	case network.RunCmdMsg:
		var cmd tea.Cmd
		a.networkModel, cmd = a.networkModel.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return a, tea.Batch(cmds...)

	case container.ContainerActionMsg:
		if msg.Err != nil {
			a.errorModal.Show("Container Action Failed", msg.Err.Error())
		} else {
			cmds = append(cmds, a.handleActionSuccess(TabContainers)...)
		}
		var cmd tea.Cmd
		a.containerModel, cmd = a.containerModel.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return a, tea.Batch(cmds...)

	case container.ComposeActionMsg:
		if msg.Err != nil {
			a.errorModal.Show("Compose Action Failed", msg.Err.Error())
		} else {
			cmds = append(cmds, a.handleActionSuccess(TabContainers)...)
		}
		var cmd tea.Cmd
		a.containerModel, cmd = a.containerModel.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return a, tea.Batch(cmds...)

	case image.ImageResultMsg:
		if msg.Err != nil {
			a.errorModal.Show("Image Error", msg.Err.Error())
		} else {
			cmds = append(cmds, a.handleActionSuccess(TabImages)...)
		}
		var cmd tea.Cmd
		a.imageModel, cmd = a.imageModel.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return a, tea.Batch(cmds...)

	case volume.VolumeResultMsg:
		if msg.Err != nil {
			a.errorModal.Show("Volume Error", msg.Err.Error())
		} else {
			cmds = append(cmds, a.handleActionSuccess(TabVolumes)...)
		}
		var cmd tea.Cmd
		a.volumeModel, cmd = a.volumeModel.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		return a, tea.Batch(cmds...)

	case network.NetworkResultMsg:
		if msg.Err != nil {
			a.errorModal.Show("Network Error", msg.Err.Error())
		} else {
			cmds = append(cmds, a.handleActionSuccess(TabNetworks)...)
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

	case container.AttachFinishedMsg:
		if msg.Err != nil {
			a.errorModal.Show("Container Attach Failed", msg.Err.Error())
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
		if a.imageModel.IsLoading() {
			var cmd tea.Cmd
			a.imageModel, cmd = a.imageModel.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			routed = true
		}
		if a.volumeModel.IsLoading() {
			var cmd tea.Cmd
			a.volumeModel, cmd = a.volumeModel.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			routed = true
		}
		if a.networkModel.IsLoading() {
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
	if a.confirmModal.Active {
		return a.confirmModal.FooterBindings()
	}
	if a.errorModal.Active {
		return a.errorModal.FooterBindings()
	}
	if a.actionMenuModal.Active {
		return a.actionMenuModal.FooterBindings()
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

	var pruningStatus string
	switch a.activeTab {
	case TabContainers:
		if a.containerModel.IsPruning() {
			pruningStatus = theme.Current.ActionLoading.Render(a.containerModel.SpinnerView() + " Pruning containers...")
		}
	case TabImages:
		if a.imageModel.IsPruning() {
			pruningStatus = theme.Current.ActionLoading.Render(a.imageModel.SpinnerView() + " Pruning images...")
		}
	case TabVolumes:
		if a.volumeModel.IsPruning() {
			pruningStatus = theme.Current.ActionLoading.Render(a.volumeModel.SpinnerView() + " Pruning volumes...")
		}
	case TabNetworks:
		if a.networkModel.IsPruning() {
			pruningStatus = theme.Current.ActionLoading.Render(a.networkModel.SpinnerView() + " Pruning networks...")
		}
	}

	rawHost := a.dockerClient.Host()
	hostLabel := theme.Current.Key.Render("Host: ")
	hostValue := theme.Current.Value.Render(rawHost)
	fullRight := hostLabel + hostValue + "  " + statusBadge
	if pruningStatus != "" {
		fullRight = pruningStatus + "  " + fullRight
	}

	leftWidth := ansi.StringWidth(headerTabs)
	rightWidth := a.width - leftWidth

	var rightView string
	if rightWidth > 0 {
		var rightContent string
		if rightWidth >= ansi.StringWidth(fullRight)+2 {
			rightContent = fullRight
		} else if pruningStatus != "" && rightWidth >= ansi.StringWidth(pruningStatus+"  "+statusBadge)+2 {
			rightContent = pruningStatus + "  " + statusBadge
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
	if a.confirmModal.Active {
		out = a.confirmModal.Overlay(out, a.width, a.height)
	} else if a.errorModal.Active {
		out = a.errorModal.Overlay(out, a.width, a.height)
	} else if a.actionMenuModal.Active {
		out = a.actionMenuModal.Overlay(out, a.width, a.height)
	}

	v := tea.NewView(out)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}
