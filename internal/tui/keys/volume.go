package keys

import "charm.land/bubbles/v2/key"

type VolumeKeyMap struct {
	Remove key.Binding
	Prune  key.Binding
	Up     key.Binding
	Down   key.Binding
	Left   key.Binding
	Right  key.Binding
	Focus  key.Binding
}

var VolumeKeys = VolumeKeyMap{
	Remove: key.NewBinding(
		key.WithKeys("d", "D", "x", "X"),
		key.WithHelp("d", "Remove"),
	),
	Prune: key.NewBinding(
		key.WithKeys("p", "P"),
		key.WithHelp("p", "Prune Unused"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("k/↑", "Up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("j/↓", "Down"),
	),
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("h/←", "Left"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("l/→", "Right"),
	),
	Focus: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("Tab", "Focus"),
	),
}
