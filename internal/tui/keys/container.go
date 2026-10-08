package keys

import "charm.land/bubbles/v2/key"

type ContainerKeyMap struct {
	StartStop    key.Binding
	Start        key.Binding
	Stop         key.Binding
	Pause        key.Binding
	Unpause      key.Binding
	Restart      key.Binding
	Remove       key.Binding
	Prune        key.Binding
	Exec         key.Binding
	NextPane     key.Binding
	PrevPane     key.Binding
	ToggleExpand key.Binding
	Collapse     key.Binding
	Expand       key.Binding
	Up           key.Binding
	Down         key.Binding
	FollowLogs   key.Binding
	ToggleRaw    key.Binding
}

var ContainerKeys = ContainerKeyMap{
	StartStop: key.NewBinding(
		key.WithKeys("s", "S"),
		key.WithHelp("s", "Start/Stop"),
	),
	Start: key.NewBinding(
		key.WithKeys("s", "S"),
		key.WithHelp("s", "Start"),
	),
	Stop: key.NewBinding(
		key.WithKeys("s", "S"),
		key.WithHelp("s", "Stop"),
	),
	Pause: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "Pause"),
	),
	Unpause: key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", "Unpause"),
	),
	Restart: key.NewBinding(
		key.WithKeys("r", "R"),
		key.WithHelp("r", "Restart"),
	),
	Remove: key.NewBinding(
		key.WithKeys("d", "D", "x", "X"),
		key.WithHelp("d", "Remove"),
	),
	Prune: key.NewBinding(
		key.WithKeys("P"),
		key.WithHelp("P", "Prune"),
	),
	Exec: key.NewBinding(
		key.WithKeys("e", "E"),
		key.WithHelp("e", "Exec"),
	),
	NextPane: key.NewBinding(
		key.WithKeys("]"),
		key.WithHelp("]", "Next Pane"),
	),
	PrevPane: key.NewBinding(
		key.WithKeys("["),
		key.WithHelp("[", "Prev Pane"),
	),
	ToggleExpand: key.NewBinding(
		key.WithKeys("space", "enter"),
		key.WithHelp("Space/Enter", "Toggle"),
	),
	Collapse: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("h/←", "Collapse"),
	),
	Expand: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("l/→", "Expand"),
	),
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("k/↑", "Up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("j/↓", "Down"),
	),
	FollowLogs: key.NewBinding(
		key.WithKeys("G"),
		key.WithHelp("G", "Follow"),
	),
	ToggleRaw: key.NewBinding(
		key.WithKeys("t", "T"),
		key.WithHelp("t", "Raw JSON"),
	),
}
