package keys

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/key"
	"github.com/xoaiPro235/docktop/internal/tui/theme"
)

func TestRenderFooterHelp(t *testing.T) {
	theme.SetThemeByName("tokyo-night")

	b1 := key.NewBinding(
		key.WithKeys("q"),
		key.WithHelp("q", "Quit"),
	)
	b2 := key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("Tab", "Focus"),
	)
	bDisabled := key.NewBinding(
		key.WithKeys("x"),
		key.WithHelp("x", "Hidden"),
		key.WithDisabled(),
	)
	bEmptyHelp := key.NewBinding(
		key.WithKeys("y"),
	)

	tests := []struct {
		name     string
		bindings []key.Binding
		wantPart []string
		notWant  string
	}{
		{
			name:     "empty bindings",
			bindings: []key.Binding{},
			wantPart: []string{" "},
		},
		{
			name:     "single binding",
			bindings: []key.Binding{b1},
			wantPart: []string{"q", "Quit"},
		},
		{
			name:     "multiple bindings with separator",
			bindings: []key.Binding{b1, b2},
			wantPart: []string{"q", "Quit", "│", "Tab", "Focus"},
		},
		{
			name:     "ignores disabled and empty help bindings",
			bindings: []key.Binding{bDisabled, bEmptyHelp, b1},
			wantPart: []string{"q", "Quit"},
			notWant:  "Hidden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderFooterHelp(tt.bindings...)
			for _, part := range tt.wantPart {
				if !strings.Contains(got, part) {
					t.Errorf("RenderFooterHelp() = %q, expected to contain %q", got, part)
				}
			}
			if tt.notWant != "" && strings.Contains(got, tt.notWant) {
				t.Errorf("RenderFooterHelp() = %q, expected NOT to contain %q", got, tt.notWant)
			}
		})
	}
}

func TestAppKeys_Bindings(t *testing.T) {
	bindings := []struct {
		name string
		b    key.Binding
	}{
		{"Quit", AppKeys.Quit},
		{"ForceQuit", AppKeys.ForceQuit},
		{"TabContainers", AppKeys.TabContainers},
		{"TabImages", AppKeys.TabImages},
		{"TabVolumes", AppKeys.TabVolumes},
		{"TabNetworks", AppKeys.TabNetworks},
		{"FocusToggle", AppKeys.FocusToggle},
	}

	for _, tc := range bindings {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.b.Enabled() {
				t.Errorf("AppKey %s is disabled", tc.name)
			}
			help := tc.b.Help()
			if help.Key == "" || help.Desc == "" {
				t.Errorf("AppKey %s has empty help: %+v", tc.name, help)
			}
		})
	}
}

func TestCommonHelpKeys_Bindings(t *testing.T) {
	bindings := []struct {
		name string
		b    key.Binding
	}{
		{"HelpMove", CommonHelpKeys.HelpMove},
		{"HelpMoveClick", CommonHelpKeys.HelpMoveClick},
		{"HelpNavigate", CommonHelpKeys.HelpNavigate},
		{"HelpScroll", CommonHelpKeys.HelpScroll},
		{"HelpScrollX", CommonHelpKeys.HelpScrollX},
		{"HelpScrollUp", CommonHelpKeys.HelpScrollUp},
		{"HelpScrollBoth", CommonHelpKeys.HelpScrollBoth},
		{"HelpSubpane", CommonHelpKeys.HelpSubpane},
		{"HelpFilter", CommonHelpKeys.HelpFilter},
		{"HelpFilterActive", CommonHelpKeys.HelpFilterActive},
	}

	for _, tc := range bindings {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.b.Enabled() {
				t.Errorf("CommonHelpKey %s is disabled", tc.name)
			}
			help := tc.b.Help()
			if help.Key == "" || help.Desc == "" {
				t.Errorf("CommonHelpKey %s has empty help: %+v", tc.name, help)
			}
		})
	}
}

func TestSearchBarKeys_Bindings(t *testing.T) {
	bindings := []struct {
		name string
		b    key.Binding
	}{
		{"Activate", SearchBarKeys.Activate},
		{"ClearFilter", SearchBarKeys.ClearFilter},
		{"Cancel", SearchBarKeys.Cancel},
		{"Submit", SearchBarKeys.Submit},
	}

	for _, tc := range bindings {
		t.Run(tc.name, func(t *testing.T) {
			if !tc.b.Enabled() {
				t.Errorf("SearchBarKey %s is disabled", tc.name)
			}
			help := tc.b.Help()
			if help.Key == "" || help.Desc == "" {
				t.Errorf("SearchBarKey %s has empty help: %+v", tc.name, help)
			}
		})
	}
}
