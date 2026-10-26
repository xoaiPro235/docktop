package searchbar

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/xoaiPro235/docktop/internal/tui/keys"
	"github.com/xoaiPro235/docktop/internal/tui/theme"
)

// Event represents an outcome of an input update.
type Event int

const (
	EventNone Event = iota
	EventChanged
	EventSubmitted
	EventCanceled
)

// Model wraps a textinput.Model configured for search/filtering.
type Model struct {
	input  textinput.Model
	active bool
	query  string
}

// New creates a new SearchBar model with default docktop styling.
func New(placeholder string) Model {
	ti := textinput.New()
	ti.Prompt = "🔍 / "
	ti.Placeholder = placeholder
	ti.CharLimit = 64

	styles := textinput.DefaultDarkStyles()
	styles.Focused.Prompt = theme.Current.FooterKey
	styles.Focused.Text = theme.Current.AccentText
	styles.Focused.Placeholder = theme.Current.MutedText
	styles.Blurred.Prompt = theme.Current.MutedText
	styles.Blurred.Text = theme.Current.Value
	styles.Blurred.Placeholder = theme.Current.MutedText
	ti.SetStyles(styles)

	return Model{
		input:  ti,
		active: false,
		query:  "",
	}
}

// Activate enables the search bar and focuses the text input.
func (m *Model) Activate() tea.Cmd {
	m.active = true
	return m.input.Focus()
}

// Deactivate closes the search input without clearing the current filter query.
func (m *Model) Deactivate() {
	m.active = false
	m.input.Blur()
}

// Clear resets the search input and clears the query.
func (m *Model) Clear() {
	m.active = false
	m.query = ""
	m.input.Reset()
	m.input.Blur()
}

// IsActive returns whether the search bar is currently accepting keyboard input.
func (m Model) IsActive() bool {
	return m.active
}

// HasFilter returns true if there is an active non-empty search query.
func (m Model) HasFilter() bool {
	return m.query != ""
}

// Value returns the current trimmed search query.
func (m Model) Value() string {
	return m.query
}

// Update processes key events for the search input.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd, Event) {
	if !m.active {
		if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
			if key.Matches(keyMsg, keys.SearchBarKeys.Activate) {
				cmd := m.Activate()
				return m, cmd, EventNone
			}
		}
		return m, nil, EventNone
	}

	if keyMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch {
		case key.Matches(keyMsg, keys.SearchBarKeys.Cancel):
			m.Clear()
			return m, nil, EventCanceled
		case key.Matches(keyMsg, keys.SearchBarKeys.Submit):
			m.Deactivate()
			return m, nil, EventSubmitted
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	newQuery := strings.TrimSpace(m.input.Value())
	if newQuery != m.query {
		m.query = newQuery
		return m, cmd, EventChanged
	}

	return m, cmd, EventNone
}

// View renders the search bar based on its active and filtered state.
func (m Model) View(width int) string {
	if !m.active && m.query == "" {
		return ""
	}

	if m.active {
		// When active, calculate remaining width for helper hint
		inputView := m.input.View()
		hint := theme.Current.MutedText.Render("[esc: cancel | enter: done]")
		inputWidth := ansi.StringWidth(inputView)
		hintWidth := ansi.StringWidth(hint)

		available := width - inputWidth - 2
		if available > hintWidth+2 {
			spaces := strings.Repeat(" ", available-hintWidth)
			return fmt.Sprintf(" %s%s%s", inputView, spaces, hint)
		}
		return " " + inputView
	}

	// Filter is applied but input is closed (inactive browsing mode)
	filterBadge := theme.Current.StatusRunning.Render(fmt.Sprintf("🔍 Filter: %q", m.query))
	hint := theme.Current.MutedText.Render("[/ to edit | esc to clear]")
	badgeWidth := ansi.StringWidth(filterBadge)
	hintWidth := ansi.StringWidth(hint)

	available := width - badgeWidth - 2
	if available > hintWidth+2 {
		spaces := strings.Repeat(" ", available-hintWidth)
		return fmt.Sprintf(" %s%s%s", filterBadge, spaces, hint)
	}
	return " " + filterBadge
}
