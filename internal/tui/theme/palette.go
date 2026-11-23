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

// DefaultPalette returns the Tokyo Night theme.
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
		TextMuted:     lipgloss.Color("#787c99"),
		TextDark:      lipgloss.Color("#565f89"),
		BorderNormal:  lipgloss.Color("#3b4261"),
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

// DraculaPalette returns the Dracula dark theme.
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
		ForegroundDim: lipgloss.Color("#dadad5"),
		TextMuted:     lipgloss.Color("#6272a4"),
		TextDark:      lipgloss.Color("#44475a"),
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

// CatppuccinMochaPalette returns the Catppuccin Mocha theme.
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
		BorderNormal:  lipgloss.Color("#45475a"),
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

// NordPalette returns the arctic, north-bluish Nord palette matching official specs.
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
		TextMuted:     lipgloss.Color("#4c566a"),
		TextDark:      lipgloss.Color("#434c5e"),
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

// OneDarkPalette returns the One Dark theme.
func OneDarkPalette() Palette {
	return Palette{
		Primary:       lipgloss.Color("#61afef"),
		Secondary:     lipgloss.Color("#c678dd"),
		Accent:        lipgloss.Color("#56b6c2"),
		Highlight:     lipgloss.Color("#d19a66"),
		Success:       lipgloss.Color("#98c379"),
		Warning:       lipgloss.Color("#e5c07b"),
		Danger:        lipgloss.Color("#e06c75"),
		Foreground:    lipgloss.Color("#abb2bf"),
		ForegroundDim: lipgloss.Color("#9da5b4"),
		TextMuted:     lipgloss.Color("#5c6370"),
		TextDark:      lipgloss.Color("#495162"),
		BorderNormal:  lipgloss.Color("#3e4452"),
		BorderFocused: lipgloss.Color("#61afef"),
		BgTabActive:   lipgloss.Color("#323842"),
		BgPanel:       lipgloss.Color("#21252b"),
		ChartGradient: []color.Color{
			lipgloss.Color("#98c379"),
			lipgloss.Color("#e5c07b"),
			lipgloss.Color("#d19a66"),
			lipgloss.Color("#e06c75"),
		},
	}
}

// GruvboxDarkPalette returns the Gruvbox dark theme with medium contrast.
func GruvboxDarkPalette() Palette {
	return Palette{
		Primary:       lipgloss.Color("#fabd2f"),
		Secondary:     lipgloss.Color("#83a598"),
		Accent:        lipgloss.Color("#8ec07c"),
		Highlight:     lipgloss.Color("#fe8019"),
		Success:       lipgloss.Color("#b8bb26"),
		Warning:       lipgloss.Color("#fabd2f"),
		Danger:        lipgloss.Color("#fb4934"),
		Foreground:    lipgloss.Color("#ebdbb2"),
		ForegroundDim: lipgloss.Color("#d5c4a1"),
		TextMuted:     lipgloss.Color("#928374"),
		TextDark:      lipgloss.Color("#665c54"),
		BorderNormal:  lipgloss.Color("#504945"),
		BorderFocused: lipgloss.Color("#fabd2f"),
		BgTabActive:   lipgloss.Color("#3c3836"),
		BgPanel:       lipgloss.Color("#282828"),
		ChartGradient: []color.Color{
			lipgloss.Color("#b8bb26"),
			lipgloss.Color("#fabd2f"),
			lipgloss.Color("#fe8019"),
			lipgloss.Color("#fb4934"),
		},
	}
}

// EverforestPalette returns the Everforest dark theme with medium contrast.
func EverforestPalette() Palette {
	return Palette{
		Primary:       lipgloss.Color("#a7c080"),
		Secondary:     lipgloss.Color("#7fbbb3"),
		Accent:        lipgloss.Color("#83c092"),
		Highlight:     lipgloss.Color("#a7c080"),
		Success:       lipgloss.Color("#a7c080"),
		Warning:       lipgloss.Color("#dbbc7f"),
		Danger:        lipgloss.Color("#e67e80"),
		Foreground:    lipgloss.Color("#d3c6aa"),
		ForegroundDim: lipgloss.Color("#9da9a0"),
		TextMuted:     lipgloss.Color("#859289"),
		TextDark:      lipgloss.Color("#7a8478"),
		BorderNormal:  lipgloss.Color("#414b50"),
		BorderFocused: lipgloss.Color("#a7c080"),
		BgTabActive:   lipgloss.Color("#374145"),
		BgPanel:       lipgloss.Color("#272e33"),
		ChartGradient: []color.Color{
			lipgloss.Color("#a7c080"),
			lipgloss.Color("#dbbc7f"),
			lipgloss.Color("#e69875"),
			lipgloss.Color("#e67e80"),
		},
	}
}

// RosePinePalette returns the Rosé Pine dark theme.
func RosePinePalette() Palette {
	return Palette{
		Primary:       lipgloss.Color("#ebbcba"),
		Secondary:     lipgloss.Color("#c4a7e7"),
		Accent:        lipgloss.Color("#9ccfd8"),
		Highlight:     lipgloss.Color("#ebbcba"),
		Success:       lipgloss.Color("#9ccfd8"),
		Warning:       lipgloss.Color("#f6c177"),
		Danger:        lipgloss.Color("#eb6f92"),
		Foreground:    lipgloss.Color("#e0def4"),
		ForegroundDim: lipgloss.Color("#908caa"),
		TextMuted:     lipgloss.Color("#908caa"),
		TextDark:      lipgloss.Color("#817c9c"),
		BorderNormal:  lipgloss.Color("#524f67"),
		BorderFocused: lipgloss.Color("#ebbcba"),
		BgTabActive:   lipgloss.Color("#2a283e"),
		BgPanel:       lipgloss.Color("#191724"),
		ChartGradient: []color.Color{
			lipgloss.Color("#9ccfd8"),
			lipgloss.Color("#f6c177"),
			lipgloss.Color("#ebbcba"),
			lipgloss.Color("#eb6f92"),
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
	"one-dark":    OneDarkPalette(),
	"gruvbox":     GruvboxDarkPalette(),
	"everforest":  EverforestPalette(),
	"rose-pine":   RosePinePalette(),
}
