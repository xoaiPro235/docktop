package keys

import "charm.land/bubbles/v2/key"

type ImageKeyMap struct {
	Remove      key.Binding
	Prune       key.Binding
	Up          key.Binding
	Down        key.Binding
	Left        key.Binding
	Right       key.Binding
	FocusToggle key.Binding
	ToggleWrap  key.Binding
}

var ImageKeys = ImageKeyMap{
	Remove: key.NewBinding(
		key.WithKeys("r", "R", "d", "D"),
		key.WithHelp("r/d", "Remove"),
	),
	Prune: key.NewBinding(
		key.WithKeys("p", "P"),
		key.WithHelp("p", "Prune"),
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
	FocusToggle: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("Tab", "Focus"),
	),
	ToggleWrap: key.NewBinding(
		key.WithKeys("w", "W"),
		key.WithHelp("w", "Wrap"),
	),
}
