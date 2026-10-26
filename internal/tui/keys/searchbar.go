package keys

import "charm.land/bubbles/v2/key"

// SearchBarKeyMap holds key bindings for search bar interaction.
type SearchBarKeyMap struct {
	Activate    key.Binding
	ClearFilter key.Binding
	Cancel      key.Binding
	Submit      key.Binding
}

// SearchBarKeys is the default instance of SearchBarKeyMap.
var SearchBarKeys = SearchBarKeyMap{
	Activate: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "Filter"),
	),
	ClearFilter: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("esc", "Clear Filter"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("esc"),
		key.WithHelp("Esc", "Cancel"),
	),
	Submit: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("Enter", "Done"),
	),
}
