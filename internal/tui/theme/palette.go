package theme

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Palette holds the semantic color tokens for the entire TUI application.
type Palette struct {
	Primary       color.Color
	Secondary     color.Color
	Accent        color.Color
	Highlight     color.Color
	Success       color.Color
	Warning       color.Color
	Danger        color.Color
	Foreground    color.Color
	ForegroundDim color.Color
	TextMuted     color.Color
	TextDark      color.Color
	BorderNormal  color.Color
	BorderFocused color.Color
	BgTabActive   color.Color
	BgPanel       color.Color
	ChartGradient []color.Color
}

// DefaultPalette returns the authentic Tokyo Night theme.
func DefaultPalette() Palette {
	return Palette{
		Primary:       lipgloss.Color("#bb9af7"),
		Secondary:     lipgloss.Color("#7aa2f7"),
		Accent:        lipgloss.Color("#7dcfff"),
		Highlight:     lipgloss.Color("#f7768e"),
		Success:       lipgloss.Color("#9ece6a"),
		Warning:       lipgloss.Color("#e0af68"),
		Danger:        lipgloss.Color("#db4b4b"),
		Foreground:    lipgloss.Color("#c0caf5"),
		ForegroundDim: lipgloss.Color("#a9b1d6"),
		TextMuted:     lipgloss.Color("#565f89"),
		TextDark:      lipgloss.Color("#545c7e"),
		BorderNormal:  lipgloss.Color("#292e42"),
		BorderFocused: lipgloss.Color("#7aa2f7"),
		BgTabActive:   lipgloss.Color("#202330"),
		BgPanel:       lipgloss.Color("#16161e"),
		ChartGradient: []color.Color{
			lipgloss.Color("#73daca"),
			lipgloss.Color("#7aa2f7"),
			lipgloss.Color("#bb9af7"),
			lipgloss.Color("#f7768e"),
		},
	}
}

// TerminalPalette returns an adaptive palette mapped to the terminal's 16 ANSI colors (0-15).
func TerminalPalette() Palette {
	return Palette{
		Primary:       lipgloss.Color("4"),
		Secondary:     lipgloss.Color("6"),
		Accent:        lipgloss.Color("14"),
		Highlight:     lipgloss.Color("5"),
		Success:       lipgloss.Color("2"),
		Warning:       lipgloss.Color("3"),
		Danger:        lipgloss.Color("1"),
		Foreground:    lipgloss.Color("15"),
		ForegroundDim: lipgloss.Color("7"),
		TextMuted:     lipgloss.Color("8"),
		TextDark:      lipgloss.Color("8"),
		BorderNormal:  lipgloss.Color("8"),
		BorderFocused: lipgloss.Color("4"),
		BgTabActive:   lipgloss.Color("8"),
		BgPanel:       lipgloss.Color("0"),
		ChartGradient: []color.Color{
			lipgloss.Color("2"),
			lipgloss.Color("3"),
			lipgloss.Color("9"),
			lipgloss.Color("1"),
		},
	}
}

// DraculaPalette returns the authentic Dracula dark theme.
func DraculaPalette() Palette {
	return Palette{
		Primary:       lipgloss.Color("#bd93f9"),
		Secondary:     lipgloss.Color("#8be9fd"),
		Accent:        lipgloss.Color("#f1fa8c"),
		Highlight:     lipgloss.Color("#ff79c6"),
		Success:       lipgloss.Color("#50fa7b"),
		Warning:       lipgloss.Color("#ffb86c"),
		Danger:        lipgloss.Color("#ff5555"),
		Foreground:    lipgloss.Color("#f8f8f2"),
		ForegroundDim: lipgloss.Color("#e2e8f0"),
		TextMuted:     lipgloss.Color("#6272a4"),
		TextDark:      lipgloss.Color("#6272a4"),
		BorderNormal:  lipgloss.Color("#44475a"),
		BorderFocused: lipgloss.Color("#bd93f9"),
		BgTabActive:   lipgloss.Color("#44475a"),
		BgPanel:       lipgloss.Color("#282a36"),
		ChartGradient: []color.Color{
			lipgloss.Color("#50fa7b"),
			lipgloss.Color("#f1fa8c"),
			lipgloss.Color("#ffb86c"),
			lipgloss.Color("#ff5555"),
		},
	}
}

// CatppuccinMochaPalette returns the authentic Catppuccin Mocha theme.
func CatppuccinMochaPalette() Palette {
	return Palette{
		Primary:       lipgloss.Color("#cba6f7"),
		Secondary:     lipgloss.Color("#89b4fa"),
		Accent:        lipgloss.Color("#94e2d5"),
		Highlight:     lipgloss.Color("#f5c2e7"),
		Success:       lipgloss.Color("#a6e3a1"),
		Warning:       lipgloss.Color("#fab387"),
		Danger:        lipgloss.Color("#f38ba8"),
		Foreground:    lipgloss.Color("#cdd6f4"),
		ForegroundDim: lipgloss.Color("#bac2de"),
		TextMuted:     lipgloss.Color("#a6adc8"),
		TextDark:      lipgloss.Color("#7f849c"),
		BorderFocused: lipgloss.Color("#cba6f7"),
		BgTabActive:   lipgloss.Color("#313244"),
		BgPanel:       lipgloss.Color("#1e1e2e"),
		ChartGradient: []color.Color{
			lipgloss.Color("#a6e3a1"),
			lipgloss.Color("#f9e2af"),
			lipgloss.Color("#fab387"),
			lipgloss.Color("#f38ba8"),
		},
	}
}

// NordPalette returns the authentic arctic, north-bluish Nord palette matching official specs.
func NordPalette() Palette {
	return Palette{
		Primary:       lipgloss.Color("#88c0d0"),
		Secondary:     lipgloss.Color("#81a1c1"),
		Accent:        lipgloss.Color("#8fbcbb"),
		Highlight:     lipgloss.Color("#b48ead"),
		Success:       lipgloss.Color("#a3be8c"),
		Warning:       lipgloss.Color("#ebcb8b"),
		Danger:        lipgloss.Color("#bf616a"),
		Foreground:    lipgloss.Color("#eceff4"),
		ForegroundDim: lipgloss.Color("#d8dee9"),
		TextMuted:     lipgloss.Color("#5e81ac"),
		TextDark:      lipgloss.Color("#5e81ac"),
		BorderNormal:  lipgloss.Color("#434c5e"),
		BorderFocused: lipgloss.Color("#88c0d0"),
		BgTabActive:   lipgloss.Color("#3b4252"),
		BgPanel:       lipgloss.Color("#2e3440"),
		ChartGradient: []color.Color{
			lipgloss.Color("#81a1c1"),
			lipgloss.Color("#88c0d0"),
			lipgloss.Color("#ebcb8b"),
			lipgloss.Color("#bf616a"),
		},
	}
}

// Presets maps theme names to their corresponding palette.
var Presets = map[string]Palette{
	"tokyo-night": DefaultPalette(),
	"terminal":    TerminalPalette(),
	"adaptive":    TerminalPalette(),
	"auto":        TerminalPalette(),
	"dracula":     DraculaPalette(),
	"catppuccin":  CatppuccinMochaPalette(),
	"nord":        NordPalette(),
}
