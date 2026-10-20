package theme

import (
	"testing"
)

func TestNewTheme(t *testing.T) {
	p := DefaultPalette()
	th := NewTheme(p)

	if th == nil {
		t.Fatal("NewTheme returned nil")
	}

	// Verify that basic styles can render without panic
	if th.TabActive.Render("Active") == "" {
		t.Error("TabActive rendered empty string")
	}
	if th.FooterKey.Render("q") == "" {
		t.Error("FooterKey rendered empty string")
	}
	if th.StatusRunning.Render("running") == "" {
		t.Error("StatusRunning rendered empty string")
	}
}

func TestSetTheme(t *testing.T) {
	p := DraculaPalette()
	SetTheme(p)

	if Current == nil {
		t.Fatal("Current theme is nil after SetTheme")
	}
}

func TestSetThemeByName(t *testing.T) {
	tests := []struct {
		name      string
		themeName string
		want      bool
	}{
		{"Tokyo Night", "tokyo-night", true},
		{"Terminal", "terminal", true},
		{"Dracula", "dracula", true},
		{"Catppuccin", "catppuccin", true},
		{"Nord", "nord", true},
		{"Uppercase / Spaces", "  NORD  ", true},
		{"Invalid Theme", "invalid-theme-xyz", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SetThemeByName(tt.themeName)
			if got != tt.want {
				t.Errorf("SetThemeByName(%q) = %v, want %v", tt.themeName, got, tt.want)
			}
		})
	}
}
