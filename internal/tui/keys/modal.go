package keys

import "charm.land/bubbles/v2/key"

type ConfirmModalKeyMap struct {
	Yes    key.Binding
	No     key.Binding
	Switch key.Binding
	Enter  key.Binding
}

var ConfirmModalKeys = ConfirmModalKeyMap{
	Yes: key.NewBinding(
		key.WithKeys("y", "Y"),
		key.WithHelp("y", "Yes"),
	),
	No: key.NewBinding(
		key.WithKeys("n", "N", "esc"),
		key.WithHelp("n/Esc", "No"),
	),
	Switch: key.NewBinding(
		key.WithKeys("tab", "left", "right", "h", "l"),
		key.WithHelp("Tab / ← / →", "Switch"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("Enter", "Select"),
	),
}

type ErrorModalKeyMap struct {
	Dismiss key.Binding
}

var ErrorModalKeys = ErrorModalKeyMap{
	Dismiss: key.NewBinding(
		key.WithKeys("y", "Y", "enter", "esc", "space", "q"),
		key.WithHelp("y / Enter / Esc", "Dismiss Error"),
	),
}
