package theme

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Palette holds the semantic color tokens for the entire TUI application.
// This structure is designed to be easily loaded/overridden from a user config file in the future.
type Palette struct {
	Primary       color.Color   // Tokyo Purple (headers, active borders, project nodes)
	Secondary     color.Color   // Tokyo Blue (service nodes, info headers)
	Accent        color.Color   // Tokyo Cyan (highlights, ports, cursor focus)
	Highlight     color.Color   // Pink/Magenta (tabs, active badges, exec badge)
	Success       color.Color   // Neon Green (running status, auto-scroll)
	Warning       color.Color   // Warm Orange (paused status, memory warnings)
	Danger        color.Color   // Crimson Red (stopped/exited status, errors)
	Foreground    color.Color   // Crisp White (bright text, titles)
	ForegroundDim color.Color   // Light Gray (secondary text, labels)
	TextMuted     color.Color   // Medium Gray (values, hints)
	TextDark      color.Color   // Dark Gray (footer, descriptions)
	BorderNormal  color.Color   // Inactive panel borders
	BorderFocused color.Color   // Active/Focused panel borders
	BgTabActive   color.Color   // Background for active tab
	BgPanel       color.Color   // Background for panels (if used)
	ChartGradient []color.Color // Colors for CPU/Mem sparklines and progress bar
}

// DefaultPalette returns the default Tokyo Night / Neon aesthetic palette.
func DefaultPalette() Palette {
	return Palette{
		Primary:       lipgloss.Color("141"), // Soft purple
		Secondary:     lipgloss.Color("75"),  // Sky blue
		Accent:        lipgloss.Color("86"),  // Vibrant cyan
		Highlight:     lipgloss.Color("205"), // Neon pink
		Success:       lipgloss.Color("82"),  // Vivid green
		Warning:       lipgloss.Color("214"), // Bright orange
		Danger:        lipgloss.Color("196"), // Clean red
		Foreground:    lipgloss.Color("255"), // Pure white
		ForegroundDim: lipgloss.Color("250"), // Light gray
		TextMuted:     lipgloss.Color("244"), // Neutral gray
		TextDark:      lipgloss.Color("241"), // Dark gray
		BorderNormal:  lipgloss.Color("240"), // Dim border
		BorderFocused: lipgloss.Color("63"),  // Purple/Blue focused border
		BgTabActive:   lipgloss.Color("236"), // Subtle dark background for active tab
		BgPanel:       lipgloss.Color("234"), // Dark panel background
		ChartGradient: []color.Color{
			lipgloss.Color("#73daca"), // Cyan
			lipgloss.Color("#7aa2f7"), // Blue
			lipgloss.Color("#bb9af7"), // Purple
			lipgloss.Color("#f7768e"), // Pink/Red
		},
	}
}

// TerminalPalette returns an adaptive palette mapped to the terminal's 16 ANSI colors (0-15).
// It automatically adopts the colors defined by the user's terminal emulator theme (Kitty, Alacritty, iTerm2, WezTerm, etc.).
func TerminalPalette() Palette {
	return Palette{
		Primary:       lipgloss.Color("4"),  // ANSI 4 Blue
		Secondary:     lipgloss.Color("6"),  // ANSI 6 Cyan
		Accent:        lipgloss.Color("14"), // ANSI 14 Bright Cyan
		Highlight:     lipgloss.Color("5"),  // ANSI 5 Magenta
		Success:       lipgloss.Color("2"),  // ANSI 2 Green
		Warning:       lipgloss.Color("3"),  // ANSI 3 Yellow
		Danger:        lipgloss.Color("1"),  // ANSI 1 Red
		Foreground:    lipgloss.Color("15"), // ANSI 15 Bright White
		ForegroundDim: lipgloss.Color("7"),  // ANSI 7 White
		TextMuted:     lipgloss.Color("8"),  // ANSI 8 Bright Black / Gray
		TextDark:      lipgloss.Color("8"),  // ANSI 8 Gray
		BorderNormal:  lipgloss.Color("8"),  // ANSI 8 Gray
		BorderFocused: lipgloss.Color("6"),  // ANSI 6 Cyan
		BgTabActive:   lipgloss.Color("8"),  // ANSI 8 Gray
		BgPanel:       lipgloss.Color("0"),  // ANSI 0 Black
		ChartGradient: []color.Color{
			lipgloss.Color("2"), // Green
			lipgloss.Color("6"), // Cyan
			lipgloss.Color("3"), // Yellow
			lipgloss.Color("1"), // Red
		},
	}
}

// DraculaPalette returns the classic Dracula dark theme palette.
func DraculaPalette() Palette {
	return Palette{
		Primary:       lipgloss.Color("#bd93f9"), // Dracula Purple
		Secondary:     lipgloss.Color("#8be9fd"), // Dracula Cyan
		Accent:        lipgloss.Color("#50fa7b"), // Dracula Green
		Highlight:     lipgloss.Color("#ff79c6"), // Dracula Pink
		Success:       lipgloss.Color("#50fa7b"), // Dracula Green
		Warning:       lipgloss.Color("#ffb86c"), // Dracula Orange
		Danger:        lipgloss.Color("#ff5555"), // Dracula Red
		Foreground:    lipgloss.Color("#f8f8f2"), // Dracula Foreground
		ForegroundDim: lipgloss.Color("#e2e2dc"),
		TextMuted:     lipgloss.Color("#6272a4"), // Dracula Comment
		TextDark:      lipgloss.Color("#44475a"), // Dracula Current Line
		BorderNormal:  lipgloss.Color("#44475a"),
		BorderFocused: lipgloss.Color("#bd93f9"),
		BgTabActive:   lipgloss.Color("#44475a"),
		BgPanel:       lipgloss.Color("#282a36"), // Dracula Background
		ChartGradient: []color.Color{
			lipgloss.Color("#50fa7b"), // Green
			lipgloss.Color("#8be9fd"), // Cyan
			lipgloss.Color("#ffb86c"), // Orange
			lipgloss.Color("#ff5555"), // Red
		},
	}
}

// CatppuccinMochaPalette returns the Catppuccin Mocha palette.
func CatppuccinMochaPalette() Palette {
	return Palette{
		Primary:       lipgloss.Color("#cba6f7"), // Mauve
		Secondary:     lipgloss.Color("#89b4fa"), // Blue
		Accent:        lipgloss.Color("#94e2d5"), // Teal
		Highlight:     lipgloss.Color("#f5c2e7"), // Pink
		Success:       lipgloss.Color("#a6e3a1"), // Green
		Warning:       lipgloss.Color("#fab387"), // Peach
		Danger:        lipgloss.Color("#f38ba8"), // Red
		Foreground:    lipgloss.Color("#cdd6f4"), // Text
		ForegroundDim: lipgloss.Color("#bac2de"), // Subtext1
		TextMuted:     lipgloss.Color("#a6adc8"), // Subtext0
		TextDark:      lipgloss.Color("#585b70"), // Surface2
		BorderNormal:  lipgloss.Color("#45475a"), // Surface1
		BorderFocused: lipgloss.Color("#cba6f7"), // Mauve
		BgTabActive:   lipgloss.Color("#313244"), // Surface0
		BgPanel:       lipgloss.Color("#1e1e2e"), // Base
		ChartGradient: []color.Color{
			lipgloss.Color("#a6e3a1"), // Green
			lipgloss.Color("#89b4fa"), // Blue
			lipgloss.Color("#fab387"), // Peach
			lipgloss.Color("#f38ba8"), // Red
		},
	}
}

// NordPalette returns the arctic, north-bluish Nord palette.
func NordPalette() Palette {
	return Palette{
		Primary:       lipgloss.Color("#88c0d0"), // Frost cyan/blue (nord8)
		Secondary:     lipgloss.Color("#81a1c1"), // Frost blue (nord9)
		Accent:        lipgloss.Color("#8fbcbb"), // Frost teal (nord7)
		Highlight:     lipgloss.Color("#b48ead"), // Aurora purple (nord15)
		Success:       lipgloss.Color("#a3be8c"), // Aurora green (nord14)
		Warning:       lipgloss.Color("#ebcb8b"), // Aurora yellow (nord13)
		Danger:        lipgloss.Color("#bf616a"), // Aurora red (nord11)
		Foreground:    lipgloss.Color("#eceff4"), // Snow Storm (nord6)
		ForegroundDim: lipgloss.Color("#d8dee9"), // Snow Storm (nord4)
		TextMuted:     lipgloss.Color("#4c566a"), // Polar Night (nord3)
		TextDark:      lipgloss.Color("#434c5e"), // Polar Night (nord2)
		BorderNormal:  lipgloss.Color("#434c5e"),
		BorderFocused: lipgloss.Color("#88c0d0"),
		BgTabActive:   lipgloss.Color("#3b4252"), // Polar Night (nord1)
		BgPanel:       lipgloss.Color("#2e3440"), // Polar Night (nord0)
		ChartGradient: []color.Color{
			lipgloss.Color("#a3be8c"), // Green
			lipgloss.Color("#88c0d0"), // Cyan
			lipgloss.Color("#ebcb8b"), // Yellow
			lipgloss.Color("#bf616a"), // Red
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
