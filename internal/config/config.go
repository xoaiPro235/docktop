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
	Host  string `yaml:"host,omitempty"`
}

// Default returns a Config struct initialized with default values.
func Default() *Config {
	return &Config{
		Theme: DefaultTheme,
		Host:  "",
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
	cfg.Host = strings.TrimSpace(cfg.Host)

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
#   - tokyo-night : Default vibrant Tokyo Night / Neon aesthetic
#   - terminal    : Adaptive 16 ANSI colors (automatically matches your terminal emulator theme: Kitty, Alacritty, iTerm2, WezTerm)
#   - dracula     : Classic Dracula dark theme
#   - catppuccin  : Soothing Catppuccin Mocha theme
#   - nord        : Arctic, north-bluish palette
theme: "tokyo-night"

# Docker Host:
# Optional Docker daemon endpoint or remote socket.
# If omitted or left empty, docktop falls back to the DOCKER_HOST environment variable,
# or the standard local Docker daemon socket (/var/run/docker.sock).
# Examples:
#   host: "unix:///var/run/docker.sock"
#   host: "unix:///run/user/1000/docker.sock" (Rootless Docker)
#   host: "ssh://user@remote-vps:22"
#   host: "tcp://192.168.1.100:2376"
# host: ""
`
	return os.WriteFile(path, []byte(template), 0o644)
}
