package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDefault(t *testing.T) {
	want := &Config{
		Theme: DefaultTheme,
		Host:  "",
	}
	if got := Default(); !reflect.DeepEqual(got, want) {
		t.Errorf("Default() = %v, want %v", got, want)
	}
}

func TestGetConfigPath(t *testing.T) {
	got, err := GetConfigPath()
	if err != nil {
		t.Fatalf("GetConfigPath() unexpected error: %v", err)
	}
	baseDir, _ := os.UserConfigDir()
	expected := filepath.Join(baseDir, ConfigDirName, ConfigFileName)
	if got != expected {
		t.Errorf("GetConfigPath() = %v, want %v", got, expected)
	}
}

func TestLoad(t *testing.T) {
	// Load should succeed without crashing, returning either user's config or default
	got, err := Load()
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("Load() returned nil config")
	}
	if got.Theme == "" {
		t.Errorf("Load() returned empty Theme, expected default theme")
	}
}

func TestLoadFromPath(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Valid custom config
	validPath := filepath.Join(tempDir, "custom.yaml")
	validContent := []byte("theme: dracula\nhost: unix:///tmp/docker.sock\n")
	if err := os.WriteFile(validPath, validContent, 0o644); err != nil {
		t.Fatalf("Failed to create valid config: %v", err)
	}

	// 2. Empty theme fallback config
	emptyThemePath := filepath.Join(tempDir, "empty_theme.yaml")
	if err := os.WriteFile(emptyThemePath, []byte("theme: \"   \"\nhost: \" tcp://127.0.0.1:2375 \"\n"), 0o644); err != nil {
		t.Fatalf("Failed to create empty theme config: %v", err)
	}

	// 3. Corrupted YAML config
	corruptPath := filepath.Join(tempDir, "corrupt.yaml")
	if err := os.WriteFile(corruptPath, []byte("theme: [invalid-yaml"), 0o644); err != nil {
		t.Fatalf("Failed to create corrupt config: %v", err)
	}

	// 4. Non-existent file path (should auto-generate template and return default)
	nonExistentPath := filepath.Join(tempDir, "subdir", "auto.yaml")

	tests := []struct {
		name    string
		path    string
		want    *Config
		wantErr bool
	}{
		{
			name: "valid custom config",
			path: validPath,
			want: &Config{
				Theme: "dracula",
				Host:  "unix:///tmp/docker.sock",
			},
			wantErr: false,
		},
		{
			name: "fallback empty theme and trimmed host",
			path: emptyThemePath,
			want: &Config{
				Theme: DefaultTheme,
				Host:  "tcp://127.0.0.1:2375",
			},
			wantErr: false,
		},
		{
			name:    "corrupted yaml error",
			path:    corruptPath,
			want:    Default(),
			wantErr: true,
		},
		{
			name:    "non-existent config generates template and returns default",
			path:    nonExistentPath,
			want:    Default(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadFromPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadFromPath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LoadFromPath() = %v, want %v", got, tt.want)
			}
		})
	}

	// Check that non-existent path was created by SaveTemplate
	if _, err := os.Stat(nonExistentPath); os.IsNotExist(err) {
		t.Errorf("LoadFromPath did not generate template file at %s", nonExistentPath)
	}
}

func TestSaveTemplate(t *testing.T) {
	tempDir := t.TempDir()
	targetPath := filepath.Join(tempDir, "test_template", "config.yaml")

	if err := SaveTemplate(targetPath); err != nil {
		t.Fatalf("SaveTemplate() error = %v", err)
	}

	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("Failed to read saved template: %v", err)
	}

	if len(content) == 0 {
		t.Errorf("Saved template file is empty")
	}
}
