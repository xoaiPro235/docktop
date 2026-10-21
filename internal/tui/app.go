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
func waitForDockerEvent(eventCh <-chan events.Message) tea.Cmd {
	return func() tea.Msg {
		if eventCh == nil {
			return nil
		}
		msg, ok := <-eventCh
		if !ok {
			return nil
		}
		return DockerEventMsg(msg)
	}
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
	{Tab: TabImages,     Title: "2: Images"},
	{Tab: TabVolumes,    Title: "3: Volumes"},
	{Tab: TabNetworks,   Title: "4: Networks"},
}

type AppModel struct {
	dockerClient   docker.Service
	containerModel container.Model
	imageModel     image.Model
	volumeModel    volume.Model
	networkModel   network.Model
	errorModal     modal.ErrorModal
	activeTab      Tab
	eventCancel    context.CancelFunc
	eventChan      <-chan events.Message
	width          int
	height         int
	ready          bool
}

func NewApp(cli docker.Service) AppModel {
	ctx, cancel := context.WithCancel(context.Background())
	eventCh, _ := cli.StreamEvents(ctx)

	return AppModel{
		dockerClient:   cli,
		containerModel: container.New(cli),
		imageModel:     image.New(cli),
		volumeModel:    volume.New(cli),
		networkModel:   network.New(cli),
		errorModal:     modal.NewErrorModal(),
		activeTab:      TabContainers,
		eventCancel:    cancel,
		eventChan:      eventCh,
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
		waitForDockerEvent(a.eventChan),
	)
}

func (a AppModel) isAnyModalActive() bool {
	return a.errorModal.Active ||
		a.containerModel.IsModalActive() ||
		a.imageModel.IsModalActive() ||
		a.volumeModel.IsModalActive() ||
		a.networkModel.IsModalActive()
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
		switch {
		case key.Matches(msg, keys.AppKeys.Quit):
			if a.eventCancel != nil {
				a.eventCancel()
			}
			return a, tea.Quit

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
		switch events.Type(msg.Type) {
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
		cmds = append(cmds, waitForDockerEvent(a.eventChan))


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
		if cmd := a.containerModel.SetContainers(msg); cmd != nil {
			cmds = append(cmds, cmd)
		}

	case ImageListMsg:
		cmd := a.imageModel.SetImages(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case VolumeListMsg:
		cmd := a.volumeModel.SetVolumes(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case NetworkListMsg:
		cmd := a.networkModel.SetNetworks(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case modal.ShowErrorMsg:
		a.errorModal.Show(msg.Title, msg.Message)
		return a, nil

	case container.ContainerActionMsg:
		if msg.Err != nil {
			a.errorModal.Show(fmt.Sprintf("Container %s Failed", msg.Action.Title()), msg.Err.Error())
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
	header := lipgloss.NewStyle().Width(a.width).Render(headerTabs)

	// 2. Tab content and contextual footer
	var body string
	var footerBindings []key.Binding

	switch a.activeTab {
	case TabContainers:
		body = a.containerModel.View()
		footerBindings = a.containerModel.FooterBindings()
	case TabImages:
		body = a.imageModel.View()
		footerBindings = a.imageModel.FooterBindings()
	case TabVolumes:
		body = a.volumeModel.View()
		footerBindings = a.volumeModel.FooterBindings()
	case TabNetworks:
		body = a.networkModel.View()
		footerBindings = a.networkModel.FooterBindings()
	}

	// 3. Dynamic Footer Bar based on active tab & subpane
	if a.errorModal.Active {
		footerBindings = a.errorModal.FooterBindings()
	}
	footerHelp := keys.RenderFooterHelp(footerBindings...)
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
