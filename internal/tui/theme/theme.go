package theme

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// Theme encapsulates all reusable Lipgloss styles built from a Palette.
type Theme struct {
	Palette Palette

	// Tabs
	TabActive   lipgloss.Style
	TabInactive lipgloss.Style

	// Borders & Layout
	BorderFocused lipgloss.Style
	BorderNormal  lipgloss.Style

	// Footer & Help
	Footer     lipgloss.Style
	FooterKey  lipgloss.Style
	FooterDesc lipgloss.Style

	// Tree
	TreeHeader           lipgloss.Style
	TreeBranch           lipgloss.Style
	TreeProject          lipgloss.Style
	TreeService          lipgloss.Style
	TreeContainerRunning lipgloss.Style
	TreeContainerStopped lipgloss.Style
	TreeSelected         lipgloss.Style

	// SubPane Tabs
	SubTabActive   lipgloss.Style
	SubTabInactive lipgloss.Style

	// Status Badges
	StatusRunning lipgloss.Style
	StatusPaused  lipgloss.Style
	StatusStopped lipgloss.Style
	ExecBadge     lipgloss.Style

	// Typography & Data display
	Title        lipgloss.Style
	SectionTitle lipgloss.Style
	Key          lipgloss.Style
	Value        lipgloss.Style
	AccentText   lipgloss.Style
	MutedText    lipgloss.Style
	ErrorText    lipgloss.Style
	Comment      lipgloss.Style
	LineNumber   lipgloss.Style

	// Actions & Spinners
	ActionLoading lipgloss.Style
}

// NewTheme creates a Theme configured with the given Palette.
func NewTheme(p Palette) *Theme {
	return &Theme{
		Palette: p,

		// Tabs
		TabActive: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.Highlight).
			Background(p.BgTabActive).
			Padding(0, 1),
		TabInactive: lipgloss.NewStyle().
			Foreground(p.TextMuted).
			Padding(0, 1),

		// Borders
		BorderFocused: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(p.BorderFocused),
		BorderNormal: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(p.BorderNormal),

		// Footer
		Footer: lipgloss.NewStyle().
			Foreground(p.TextDark),
		FooterKey: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.Highlight),
		FooterDesc: lipgloss.NewStyle().
			Foreground(p.TextDark),

		// Tree
		TreeHeader: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.Foreground),
		TreeBranch: lipgloss.NewStyle().
			Foreground(p.BorderNormal),
		TreeProject: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.Primary),
		TreeService: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.Secondary),
		TreeContainerRunning: lipgloss.NewStyle().
			Foreground(p.Foreground),
		TreeContainerStopped: lipgloss.NewStyle().
			Foreground(p.TextMuted),
		TreeSelected: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.Accent),

		// Sub-tabs
		SubTabActive: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.Highlight),
		SubTabInactive: lipgloss.NewStyle().
			Foreground(p.BorderNormal),

		// Status Badges
		StatusRunning: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.Success),
		StatusPaused: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.Warning),
		StatusStopped: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.Danger),
		ExecBadge: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.Highlight),

		// Typography
		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.Foreground),
		SectionTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.Secondary),
		Key: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.ForegroundDim),
		Value: lipgloss.NewStyle().
			Foreground(p.TextMuted),
		AccentText: lipgloss.NewStyle().
			Foreground(p.Accent),
		MutedText: lipgloss.NewStyle().
			Foreground(p.TextDark),
		ErrorText: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.Danger).
			Padding(1),
		Comment: lipgloss.NewStyle().
			Foreground(p.TextMuted).
			Italic(true),
		LineNumber: lipgloss.NewStyle().
			Foreground(p.BorderNormal),

		// Action loading / spinner
		ActionLoading: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.Highlight),
	}
}

// RenderStatusBadge returns a styled status string (e.g. ● RUNNING, ○ PAUSED, ○ EXITED).
func (t *Theme) RenderStatusBadge(state string) string {
	switch strings.ToLower(state) {
	case "running":
		return t.StatusRunning.Render("● RUNNING")
	case "paused":
		return t.StatusPaused.Render("○ PAUSED")
	case "restarting":
		return t.StatusPaused.Render("↻ RESTARTS")
	default:
		return t.StatusStopped.Render("○ " + strings.ToUpper(state))
	}
}

// Global active theme instance
var Current = NewTheme(DefaultPalette())

// SetTheme updates the global theme with a new palette.
func SetTheme(p Palette) {
	Current = NewTheme(p)
}

// SetThemeByName updates the global theme using a preset name (e.g. "tokyo-night", "terminal", "dracula", "nord", "catppuccin").
// Returns true if the preset was found and applied, false otherwise.
func SetThemeByName(name string) bool {
	if p, ok := Presets[strings.ToLower(strings.TrimSpace(name))]; ok {
		SetTheme(p)
		return true
	}
	return false
}
