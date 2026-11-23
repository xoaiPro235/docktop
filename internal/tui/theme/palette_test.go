package theme

import (
	"testing"
)

func assertPaletteValid(t *testing.T, name string, p Palette) {
	t.Helper()
	if p.Primary == nil {
		t.Errorf("%s: Primary color is nil", name)
	}
	if p.Secondary == nil {
		t.Errorf("%s: Secondary color is nil", name)
	}
	if p.Accent == nil {
		t.Errorf("%s: Accent color is nil", name)
	}
	if p.BorderNormal == nil {
		t.Errorf("%s: BorderNormal color is nil", name)
	}
	if p.BorderFocused == nil {
		t.Errorf("%s: BorderFocused color is nil", name)
	}
	if p.Foreground == nil {
		t.Errorf("%s: Foreground color is nil", name)
	}
	if p.Success == nil {
		t.Errorf("%s: Success color is nil", name)
	}
	if p.Danger == nil {
		t.Errorf("%s: Danger color is nil", name)
	}
	if len(p.ChartGradient) == 0 {
		t.Errorf("%s: ChartGradient is empty", name)
	}
}

func TestDefaultPalette(t *testing.T) {
	p := DefaultPalette()
	assertPaletteValid(t, "DefaultPalette", p)
}

func TestTerminalPalette(t *testing.T) {
	p := TerminalPalette()
	assertPaletteValid(t, "TerminalPalette", p)
}

func TestDraculaPalette(t *testing.T) {
	p := DraculaPalette()
	assertPaletteValid(t, "DraculaPalette", p)
}

func TestCatppuccinMochaPalette(t *testing.T) {
	p := CatppuccinMochaPalette()
	assertPaletteValid(t, "CatppuccinMochaPalette", p)
}

func TestNordPalette(t *testing.T) {
	p := NordPalette()
	assertPaletteValid(t, "NordPalette", p)
}

func TestOneDarkPalette(t *testing.T) {
	p := OneDarkPalette()
	assertPaletteValid(t, "OneDarkPalette", p)
}

func TestGruvboxDarkPalette(t *testing.T) {
	p := GruvboxDarkPalette()
	assertPaletteValid(t, "GruvboxDarkPalette", p)
}

func TestEverforestPalette(t *testing.T) {
	p := EverforestPalette()
	assertPaletteValid(t, "EverforestPalette", p)
}

func TestRosePinePalette(t *testing.T) {
	p := RosePinePalette()
	assertPaletteValid(t, "RosePinePalette", p)
}

func TestPresetsMap(t *testing.T) {
	expectedThemes := []string{
		"tokyo-night",
		"terminal",
		"dracula",
		"catppuccin",
		"nord",
		"one-dark",
		"gruvbox",
		"everforest",
		"rose-pine",
	}

	for _, name := range expectedThemes {
		p, exists := Presets[name]
		if !exists {
			t.Errorf("Preset %q not found in Presets map", name)
			continue
		}
		assertPaletteValid(t, "Preset "+name, p)
	}
}
