package keys

import (
	"strings"

	"charm.land/bubbles/v2/key"
	"github.com/xoaiPro235/docktop/internal/tui/theme"
)

type AppKeyMap struct {
	Quit          key.Binding
	TabContainers key.Binding
	TabImages     key.Binding
	TabVolumes    key.Binding
	TabNetworks   key.Binding
	FocusToggle   key.Binding
}

type CommonHelpKeyMap struct {
	HelpMove       key.Binding
	HelpMoveClick  key.Binding
	HelpNavigate   key.Binding
	HelpScroll     key.Binding
	HelpScrollX    key.Binding
	HelpScrollUp   key.Binding
	HelpScrollBoth key.Binding
	HelpSubpane    key.Binding
}

var AppKey = AppKeyMap{
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "Quit"),
	),
	TabContainers: key.NewBinding(
		key.WithKeys("1"),
		key.WithHelp("1", "Containers"),
	),
	TabImages: key.NewBinding(
		key.WithKeys("2"),
		key.WithHelp("2", "Images"),
	),
	TabVolumes: key.NewBinding(
		key.WithKeys("3"),
		key.WithHelp("3", "Volumes"),
	),
	TabNetworks: key.NewBinding(
		key.WithKeys("4"),
		key.WithHelp("4", "Networks"),
	),
	FocusToggle: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("Tab", "Toggle Focus"),
	),
}

var CommonHelpKeys = CommonHelpKeyMap{
	HelpMove: key.NewBinding(
		key.WithKeys("j", "k"),
		key.WithHelp("j/k", "Move"),
	),
	HelpMoveClick: key.NewBinding(
		key.WithKeys("j", "k"),
		key.WithHelp("j/k/Click", "Move"),
	),
	HelpNavigate: key.NewBinding(
		key.WithKeys("j", "k"),
		key.WithHelp("j/k", "Navigate"),
	),
	HelpScroll: key.NewBinding(
		key.WithKeys("j", "k"),
		key.WithHelp("j/k", "Scroll"),
	),
	HelpScrollX: key.NewBinding(
		key.WithKeys("h", "l"),
		key.WithHelp("h/l", "Scroll X"),
	),
	HelpScrollUp: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "Scroll"),
	),
	HelpScrollBoth: key.NewBinding(
		key.WithKeys("up", "down"),
		key.WithHelp("↑/↓", "Scroll"),
	),
	HelpSubpane: key.NewBinding(
		key.WithKeys("[", "]"),
		key.WithHelp("[ / ]", "Subpane"),
	),
}

func RenderFooterHelp(bindings ...key.Binding) string {
	var parts []string
	for _, b := range bindings {
		if !b.Enabled() {
			continue
		}
		h := b.Help()
		if h.Key == "" || h.Desc == "" {
			continue
		}

		keyString := theme.Current.FooterKey.Render(h.Key)
		descString := theme.Current.FooterDesc.Render(h.Desc)
		parts = append(parts, keyString+" "+descString)
	}
	return " " + strings.Join(parts, " | ")
}
