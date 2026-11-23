package theme

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
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

	// Menu & Action Menu
	MenuTitle      lipgloss.Style
	MenuCategory   lipgloss.Style
	MenuKeyBadge   lipgloss.Style
	MenuItemNormal lipgloss.Style
	MenuItemActive lipgloss.Style
	MenuCursor     lipgloss.Style

	// Modal shared
	CmdPrompt lipgloss.Style
	HintText  lipgloss.Style

	// Buttons
	BtnPrimary  lipgloss.Style
	BtnDanger   lipgloss.Style
	BtnInactive lipgloss.Style

	// Content helpers
	WarningText     lipgloss.Style
	YamlKey         lipgloss.Style
	MemSectionTitle lipgloss.Style
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
			Foreground(p.TextMuted),

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
			Foreground(p.ForegroundDim),
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

		// Menu & Action Menu
		MenuTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.Accent),
		MenuCategory: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.TextMuted),
		MenuKeyBadge: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.Primary),
		MenuItemNormal: lipgloss.NewStyle().
			Foreground(p.Foreground),
		MenuItemActive: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.Accent),
		MenuCursor: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.Accent),

		// Modal shared
		CmdPrompt: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.Accent),
		HintText: lipgloss.NewStyle().
			Foreground(p.TextDark),

		// Buttons
		BtnPrimary: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.Foreground).
			Background(p.Success).
			Padding(0, 3),
		BtnDanger: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.Foreground).
			Background(p.Danger).
			Padding(0, 3),
		BtnInactive: lipgloss.NewStyle().
			Foreground(p.ForegroundDim).
			Background(p.BorderNormal).
			Padding(0, 3),

		// Content helpers
		WarningText: lipgloss.NewStyle().
			Foreground(p.Warning),
		YamlKey: lipgloss.NewStyle().
			Foreground(p.Secondary),
		MemSectionTitle: lipgloss.NewStyle().
			Bold(true).
			Foreground(p.Warning),
	}
}

// Global active theme instance
var Current = NewTheme(DefaultPalette())

// SetTheme updates the global theme with a new palette.
func SetTheme(p Palette) {
	Current = NewTheme(p)
}

// SetThemeByName updates the global theme using a preset name (e.g. "tokyo-night", "terminal", "dracula", "nord", "catppuccin", "one-dark", "gruvbox", "everforest", "rose-pine").
// Returns true if the preset was found and applied, false otherwise.
func SetThemeByName(name string) bool {
	if p, ok := Presets[strings.ToLower(strings.TrimSpace(name))]; ok {
		SetTheme(p)
		return true
	}
	return false
}

// PaneBorder returns the appropriate border style based on whether the pane is focused.
func (t Theme) PaneBorder(focused bool) lipgloss.Style {
	if focused {
		return t.BorderFocused
	}
	return t.BorderNormal
}

// SectionHeader renders a standardized divider line with a section title (e.g. "── TITLE ──────\n").
func (t Theme) SectionHeader(title string, width int) string {
	header := fmt.Sprintf("── %s ──", title)
	if fill := width - ansi.StringWidth(header) - 2; fill > 0 {
		header += strings.Repeat("─", fill)
	}
	return t.SectionTitle.Render(header) + "\n"
}
