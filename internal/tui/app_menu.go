package tui

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"text/template"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/xoaiPro235/docktop/internal/docker"
	"github.com/xoaiPro235/docktop/internal/tui/components/container"
	"github.com/xoaiPro235/docktop/internal/tui/keys"
	"github.com/xoaiPro235/docktop/internal/tui/modal"
)

// sendKey creates an ActionItem directly from an existing key.Binding definition.
// When triggered, it dispatches the key press event to the active component.
func sendKey(b key.Binding) modal.ActionItem {
	help := b.Help()
	k := help.Key
	if len(b.Keys()) > 0 {
		k = b.Keys()[0]
	}

	var action tea.Cmd
	if len(k) > 0 {
		action = func() tea.Msg {
			return tea.KeyPressMsg{Code: rune(k[0]), Text: k}
		}
	}

	return modal.ActionItem{
		Key:    help.Key,
		Title:  help.Desc,
		Action: action,
	}
}

func getImageDisplayName(img docker.ImageInfo) string {
	if img.Tag != "" && img.Tag != "<none>:<none>" {
		return img.Tag
	}
	if len(img.RepoTags) > 0 && img.RepoTags[0] != "<none>:<none>" {
		return img.RepoTags[0]
	}
	if len(img.ID) > 12 {
		return img.ID[:12]
	}
	return img.ID
}

// openActionMenu builds the contextual action menu for the active tab and opens the modal.
func (a *AppModel) openActionMenu() tea.Cmd {
	var title string
	var actions []modal.ActionItem
	var customCmds []modal.ActionItem

	switch a.activeTab {
	case TabContainers:
		title, actions, customCmds = a.buildContainerActionMenu()
	case TabImages:
		title, actions, customCmds = a.buildImageActionMenu()
	case TabVolumes:
		title, actions, customCmds = a.buildVolumeActionMenu()
	case TabNetworks:
		title, actions, customCmds = a.buildNetworkActionMenu()
	}

	if len(actions) == 0 && len(customCmds) == 0 {
		return nil
	}

	a.actionMenuModal.Show(title, actions, customCmds, a.buildAppActions())
	return nil
}

func (a *AppModel) buildContainerActionMenu() (string, []modal.ActionItem, []modal.ActionItem) {
	var actions []modal.ActionItem
	var customCmds []modal.ActionItem
	selectedNode := a.containerModel.SelectedNode()
	selectedContainer := a.containerModel.SelectedContainer()

	title := "Container Actions"

	if selectedContainer != nil {
		title = fmt.Sprintf("Actions (%s)", selectedContainer.Name)
		ck := keys.ContainerKeys

		switch selectedContainer.State {
		case docker.StateRunning:
			actions = append(actions,
				sendKey(ck.Stop),
				sendKey(ck.Restart),
				sendKey(ck.Pause),
				sendKey(ck.Exec),
				sendKey(ck.Attach),
				sendKey(ck.OpenBrowser),
				sendKey(ck.Remove),
				sendKey(ck.Prune),
			)
		case docker.StatePaused:
			actions = append(actions,
				sendKey(ck.Unpause),
				sendKey(ck.Stop),
				sendKey(ck.Restart),
				sendKey(ck.Remove),
				sendKey(ck.Prune),
			)
		default: // stopped, exited, created
			actions = append(actions,
				sendKey(ck.Start),
				sendKey(ck.Restart),
				sendKey(ck.Remove),
				sendKey(ck.Prune),
			)
		}

		// Custom Commands for containers
		if a.cfg != nil && len(a.cfg.CustomCommands.Containers) > 0 {
			cCopy := *selectedContainer
			for i, cc := range a.cfg.CustomCommands.Containers {
				cmdDef := cc
				keyBadge := fmt.Sprintf("%d", i+1)
				rendered, renderErr := RenderCustomCommand(cmdDef.Command, cCopy)
				customCmds = append(customCmds, modal.ActionItem{
					Key:     keyBadge,
					Title:   cmdDef.Name,
					Command: rendered,
					Action: func() tea.Msg {
						if renderErr != nil {
							return modal.ShowErrorMsg{
								Title:   "Custom Command Template Error",
								Message: renderErr.Error(),
							}
						}
						return modal.ShowConfirmMsg{
							Question:  fmt.Sprintf("Run custom command %q on container %s?", cmdDef.Name, cCopy.Name),
							Command:   rendered,
							Style:     modal.StyleWarning,
							OnConfirm: ExecCustomCommand(rendered, cmdDef.Pause),
						}
					},
				})
			}
		}
	} else if selectedNode != nil && selectedNode.IsCompose() {
		title = fmt.Sprintf("Compose Actions (%s)", selectedNode.Name)
		ck := keys.ComposeKeys

		startOrStop := ck.Start
		if selectedNode.HasRunningContainers() {
			startOrStop = ck.Stop
		}

		if selectedNode.Type == container.NodeProject {
			actions = append(actions,
				sendKey(ck.Up),
				sendKey(ck.Build),
				sendKey(ck.Restart),
				sendKey(startOrStop),
				sendKey(ck.Down),
			)
		} else {
			actions = append(actions,
				sendKey(ck.Up),
				sendKey(ck.Build),
				sendKey(ck.Restart),
				sendKey(startOrStop),
				sendKey(ck.Remove),
			)
		}
	}

	return title, actions, customCmds
}

func (a *AppModel) buildImageActionMenu() (string, []modal.ActionItem, []modal.ActionItem) {
	selected := a.imageModel.SelectedImage()
	title := "Image Actions"
	actions := []modal.ActionItem{
		sendKey(keys.ImageKeys.Remove),
		sendKey(keys.ImageKeys.Prune),
	}
	var customCmds []modal.ActionItem

	if selected != nil {
		imgName := getImageDisplayName(*selected)
		title = fmt.Sprintf("Actions (%s)", imgName)

		if a.cfg != nil && len(a.cfg.CustomCommands.Images) > 0 {
			imgCopy := *selected
			for i, cc := range a.cfg.CustomCommands.Images {
				cmdDef := cc
				keyBadge := fmt.Sprintf("%d", i+1)
				rendered, renderErr := RenderCustomCommand(cmdDef.Command, imgCopy)
				customCmds = append(customCmds, modal.ActionItem{
					Key:     keyBadge,
					Title:   cmdDef.Name,
					Command: rendered,
					Action: func() tea.Msg {
						if renderErr != nil {
							return modal.ShowErrorMsg{
								Title:   "Custom Command Template Error",
								Message: renderErr.Error(),
							}
						}
						return modal.ShowConfirmMsg{
							Question:  fmt.Sprintf("Run custom command %q on image %s?", cmdDef.Name, imgName),
							Command:   rendered,
							Style:     modal.StyleWarning,
							OnConfirm: ExecCustomCommand(rendered, cmdDef.Pause),
						}
					},
				})
			}
		}
	}

	return title, actions, customCmds
}

func (a *AppModel) buildVolumeActionMenu() (string, []modal.ActionItem, []modal.ActionItem) {
	selected := a.volumeModel.SelectedVolume()
	title := "Volume Actions"
	actions := []modal.ActionItem{
		sendKey(keys.VolumeKeys.Remove),
		sendKey(keys.VolumeKeys.Prune),
	}
	var customCmds []modal.ActionItem

	if selected != nil {
		title = fmt.Sprintf("Actions (%s)", selected.Name)

		if a.cfg != nil && len(a.cfg.CustomCommands.Volumes) > 0 {
			volCopy := *selected
			for i, cc := range a.cfg.CustomCommands.Volumes {
				cmdDef := cc
				keyBadge := fmt.Sprintf("%d", i+1)
				rendered, renderErr := RenderCustomCommand(cmdDef.Command, volCopy)
				customCmds = append(customCmds, modal.ActionItem{
					Key:     keyBadge,
					Title:   cmdDef.Name,
					Command: rendered,
					Action: func() tea.Msg {
						if renderErr != nil {
							return modal.ShowErrorMsg{
								Title:   "Custom Command Template Error",
								Message: renderErr.Error(),
							}
						}
						return modal.ShowConfirmMsg{
							Question:  fmt.Sprintf("Run custom command %q on volume %s?", cmdDef.Name, volCopy.Name),
							Command:   rendered,
							Style:     modal.StyleWarning,
							OnConfirm: ExecCustomCommand(rendered, cmdDef.Pause),
						}
					},
				})
			}
		}
	}

	return title, actions, customCmds
}

func (a *AppModel) buildNetworkActionMenu() (string, []modal.ActionItem, []modal.ActionItem) {
	selected := a.networkModel.SelectedNetwork()
	title := "Network Actions"
	actions := []modal.ActionItem{
		sendKey(keys.NetworkKeys.Delete),
		sendKey(keys.NetworkKeys.Prune),
	}
	var customCmds []modal.ActionItem

	if selected != nil {
		title = fmt.Sprintf("Actions (%s)", selected.Name)

		if a.cfg != nil && len(a.cfg.CustomCommands.Networks) > 0 {
			netCopy := *selected
			for i, cc := range a.cfg.CustomCommands.Networks {
				cmdDef := cc
				keyBadge := fmt.Sprintf("%d", i+1)
				rendered, renderErr := RenderCustomCommand(cmdDef.Command, netCopy)
				customCmds = append(customCmds, modal.ActionItem{
					Key:     keyBadge,
					Title:   cmdDef.Name,
					Command: rendered,
					Action: func() tea.Msg {
						if renderErr != nil {
							return modal.ShowErrorMsg{
								Title:   "Custom Command Template Error",
								Message: renderErr.Error(),
							}
						}
						return modal.ShowConfirmMsg{
							Question:  fmt.Sprintf("Run custom command %q on network %s?", cmdDef.Name, netCopy.Name),
							Command:   rendered,
							Style:     modal.StyleWarning,
							OnConfirm: ExecCustomCommand(rendered, cmdDef.Pause),
						}
					},
				})
			}
		}
	}

	return title, actions, customCmds
}

func (a *AppModel) buildAppActions() []modal.ActionItem {
	ak := keys.AppKeys
	return []modal.ActionItem{
		sendKey(ak.FocusToggle),
		sendKey(ak.ZoomToggle),
		sendKey(ak.Reconnect),
		sendKey(ak.Quit),
	}
}

// CustomCommandFinishedMsg is emitted when a custom command process exits.
type CustomCommandFinishedMsg struct {
	Err error
}

// RenderCustomCommand replaces template placeholders like {{ .ID }} or {{ .Name }} with entity data.
func RenderCustomCommand(cmdTemplate string, data any) (string, error) {
	tmpl, err := template.New("custom_cmd").Parse(cmdTemplate)
	if err != nil {
		return "", fmt.Errorf("invalid command template %q: %w", cmdTemplate, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to evaluate command template: %w", err)
	}

	return buf.String(), nil
}

// BuildCustomCommandExec constructs the *exec.Cmd to be executed.
// If pause is true, the command is wrapped in a shell invocation that waits for user input before exiting,
// allowing the user to view command output before returning to docktop.
func BuildCustomCommandExec(renderedCmd string, pause bool) *exec.Cmd {
	trimmed := strings.TrimSpace(renderedCmd)
	if trimmed == "" {
		return nil
	}

	if pause {
		if runtime.GOOS == "windows" {
			return exec.Command("cmd.exe", "/c", trimmed+" & echo. & pause")
		}
		// Unix (Linux, macOS, BSD)
		return exec.Command("sh", "-c", trimmed+`; printf "\n\033[1;36m[Press Enter to return to docktop]\033[0m "; read -r _`)
	}

	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return nil
	}
	return exec.Command(parts[0], parts[1:]...)
}

// ExecCustomCommand executes the custom command using tea.ExecProcess.
// When pause is true, it prompts the user to press Enter before returning to docktop.
func ExecCustomCommand(renderedCmd string, pause bool) tea.Cmd {
	cmd := BuildCustomCommandExec(renderedCmd, pause)
	if cmd == nil {
		return nil
	}

	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return CustomCommandFinishedMsg{Err: err}
	})
}
