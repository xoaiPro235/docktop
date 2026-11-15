package keys

import "charm.land/bubbles/v2/key"

// ComposeKeyMap holds key bindings for Docker Compose Project and Service nodes.
type ComposeKeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Restart key.Binding
	Build   key.Binding
	Stop    key.Binding
	Start   key.Binding
	Remove  key.Binding
}

// ComposeKeys is the default instance of ComposeKeyMap.
var ComposeKeys = ComposeKeyMap{
	Up: key.NewBinding(
		key.WithKeys("u", "U"),
		key.WithHelp("u", "Up"),
	),
	Down: key.NewBinding(
		key.WithKeys("d", "D"),
		key.WithHelp("d", "Down"),
	),
	Restart: key.NewBinding(
		key.WithKeys("r", "R"),
		key.WithHelp("r", "Restart"),
	),
	Build: key.NewBinding(
		key.WithKeys("b", "B"),
		key.WithHelp("b", "Build"),
	),
	Stop: key.NewBinding(
		key.WithKeys("s", "S"),
		key.WithHelp("s", "Stop"),
	),
	Start: key.NewBinding(
		key.WithKeys("s", "S"),
		key.WithHelp("s", "Start"),
	),
	Remove: key.NewBinding(
		key.WithKeys("x", "X"),
		key.WithHelp("x", "Remove"),
	),
}
