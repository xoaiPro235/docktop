package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	DefaultTheme   = "tokyo-night"
	ConfigFileName = "config.yaml"
	ConfigDirName  = "docktop"
)

// Config represents the application configuration.
type Config struct {
	Theme string `yaml:"theme"`
}

// Default returns a Config struct initialized with default values.
func Default() *Config {
	return &Config{
		Theme: DefaultTheme,
	}
}

// GetConfigPath returns the absolute path to ~/.config/docktop/config.yaml (or OS equivalent).
func GetConfigPath() (string, error) {
	baseDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(baseDir, ConfigDirName, ConfigFileName), nil
}

// Load loads the configuration from the user's config directory.
// If the file does not exist, it automatically creates a template config file and returns the defaults.
func Load() (*Config, error) {
	path, err := GetConfigPath()
	if err != nil {
		return Default(), nil
	}
	return LoadFromPath(path)
}

// LoadFromPath reads and unmarshals the configuration from the specified file path.
func LoadFromPath(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			cfg := Default()
			_ = SaveTemplate(path)
			return cfg, nil
		}
		return Default(), err
	}

	cfg := Default()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return Default(), fmt.Errorf("failed to parse config at %s: %w", path, err)
	}

	cfg.Theme = strings.TrimSpace(cfg.Theme)
	if cfg.Theme == "" {
		cfg.Theme = DefaultTheme
	}

	return cfg, nil
}

// SaveTemplate creates a documented default config.yaml at the given path.
func SaveTemplate(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	template := `# docktop configuration file
# Location: ~/.config/docktop/config.yaml

# UI Theme:
# Available presets:
#   - tokyo-night : Tokyo Night theme (default)
#   - terminal    : Adaptive 16 ANSI colors (automatically matches your terminal emulator theme: Kitty, Alacritty, iTerm2, WezTerm)
#   - dracula     : Dracula dark theme
#   - catppuccin  : Catppuccin Mocha theme
#   - nord        : Arctic theme
theme: "tokyo-night"
`
	return os.WriteFile(path, []byte(template), 0o644)
}