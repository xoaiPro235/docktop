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

// CustomCommand represents a user-defined shell command in the configuration.
type CustomCommand struct {
	Name    string `yaml:"name"`
	Command string `yaml:"command"`
	Pause   bool   `yaml:"pause,omitempty"`
}

// CustomCommandsConfig holds custom commands categorized by target tab.
type CustomCommandsConfig struct {
	Containers []CustomCommand `yaml:"containers,omitempty"`
	Images     []CustomCommand `yaml:"images,omitempty"`
	Volumes    []CustomCommand `yaml:"volumes,omitempty"`
	Networks   []CustomCommand `yaml:"networks,omitempty"`
}

// Config represents the application configuration.
type Config struct {
	Theme          string               `yaml:"theme"`
	Host           string               `yaml:"host,omitempty"`
	Icons          map[string]string    `yaml:"icons,omitempty"`
	CustomCommands CustomCommandsConfig `yaml:"customCommands,omitempty"`
}

// DefaultIcons returns a default mapping of image/service keywords to Nerd Font symbols.
func DefaultIcons() map[string]string {
	return map[string]string{
		// Databases
		"postgres":      "",
		"postgresql":    "",
		"mysql":         "",
		"mariadb":       "",
		"mssql":         "",
		"sqlserver":     "",
		"redis":         "",
		"valkey":        "",
		"memcached":     "",
		"mongo":         "",
		"mongodb":       "",
		"sqlite":        "",
		"clickhouse":    "",
		"influxdb":      "",
		"timescaledb":   "",
		"couchdb":       "",
		"neo4j":         "",
		"cassandra":     "",
		"elasticsearch": "",
		"pgadmin":       "",
		"phpmyadmin":    "",
		"db":            "",
		"database":      "",
		"sql":           "",

		// Web servers, Reverse Proxies & Ingress
		"nginx":   "",
		"apache":  "",
		"traefik": "",
		"envoy":   "",

		// Languages & Runtimes
		"node":   "",
		"nodejs": "",
		"bun":    "",
		"deno":   "",
		"python": "",
		"golang": "",
		"go":     "",
		"rust":   "",
		"java":   "",
		"dotnet": "",
		"aspnet": "",
		"php":    "",
		"ruby":   "",
		"rails":  "",
		"elixir": "",
		"erlang": "",
		"cpp":    "",
		"swift":  "",
		"kotlin": "",
		"npm":    "",

		// Message brokers, Event streaming & Queues
		"kafka":    "󱀏",
		"rabbitmq": "",
		"nats":     "",

		// Monitoring, Logging, Tracing & Observability
		"prometheus":    "",
		"grafana":       "",
		"loki":          "",
		"cadvisor":      "",
		"jaeger":        "",
		"opentelemetry": "",
		"kibana":        "",
		"logstash":      "",
		"sentry":        "",

		// CI/CD, Git, DevOps & Automation
		"git":       "",
		"jenkins":   "",
		"gitlab":    "",
		"gitea":     "",
		"forgejo":   "",
		"argocd":    "",
		"ansible":   "",
		"terraform": "",
		"vault":     "",
		"consul":    "",
		"nomad":     "",
		"airflow":   "",
		"portainer": "",

		// Auth, Identity & Security
		"wireguard":   "",
		"cloudflared": "",

		// CMS, Apps, Media & AI / Self-hosted
		"wordpress": "",
		"ghost":     "",
		"supabase":  "",
		"appwrite":  "",
		"plex":      "󰚺",
		"emby":      "󰚴",
		"ffmpeg":    "",
		"jupyter":   "",
		"ai":        "󰧑",
		"llm":       "󰧑",
		"bot":       "󰚩",
		"mailpit":   "",
		"mailhog":   "",

		// Base OS & Shell / Utilities
		"alpine": "",
		"ubuntu": "",
		"debian": "",
		"fedora": "",
		"centos": "",
		"rocky":  "",
		"arch":   "󰣇",
		"bash":   "",
		"sh":     "",
		"zsh":    "",
		"fish":   "󰡨",

		// Tools & Fallback
		"docker":  "󰡨",
		"default": "",
	}
}

// Default returns a Config struct initialized with default values.
func Default() *Config {
	return &Config{
		Theme:          DefaultTheme,
		Host:           "",
		Icons:          DefaultIcons(),
		CustomCommands: CustomCommandsConfig{},
	}
}

// GetConfigPath returns the absolute path to the configuration file
// (~/.config/docktop/config.yaml on Linux, %APPDATA%\docktop\config.yaml on Windows,
// or ~/Library/Application Support/docktop/config.yaml on macOS).
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

	// Merge user icons over default icons
	if cfg.Icons == nil {
		cfg.Icons = DefaultIcons()
	} else {
		merged := DefaultIcons()
		for k, v := range cfg.Icons {
			merged[k] = v
		}
		cfg.Icons = merged
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
# Default locations:
#   - Linux/Unix : ~/.config/docktop/config.yaml
#   - macOS      : ~/Library/Application Support/docktop/config.yaml
#   - Windows    : %APPDATA%\docktop\config.yaml

# UI Theme:
# Available presets:
#   - tokyo-night : Default vibrant Tokyo Night / Neon aesthetic
#   - terminal    : Adaptive 16 ANSI colors (automatically matches your terminal emulator theme: Kitty, Alacritty, iTerm2, WezTerm)
#   - dracula     : Classic Dracula dark theme
#   - catppuccin  : Soothing Catppuccin Mocha theme
#   - nord        : Arctic, north-bluish palette
#   - one-dark    : Iconic Atom / VS Code One Dark Pro theme
#   - gruvbox     : Warm retro Gruvbox dark theme (medium contrast)
#   - everforest  : Natural comfortable green Everforest dark theme (medium contrast)
#   - rose-pine   : Aesthetic pastel Rosé Pine dark theme
theme: "tokyo-night"

# Docker Host:
# Optional Docker daemon endpoint or remote socket.
# If omitted or left empty, docktop falls back to the DOCKER_HOST environment variable,
# or the standard OS Docker daemon socket / named pipe.
# Examples:
#   host: "unix:///var/run/docker.sock"
#   host: "unix:///run/user/1000/docker.sock" (Rootless Docker)
#   host: "npipe:////./pipe/dockerDesktopLinuxEngine" (Windows Docker Desktop)
#   host: "ssh://user@remote-vps:22"
#   host: "tcp://192.168.1.100:2376"
# host: ""

# Container Icons:
# Maps image names, full container names, or keywords to custom Nerd Font glyphs in the container tree.
# Docktop includes standard built-in icons for common databases, web servers, and runtimes.
# You can customize or add your own mappings below:
# Examples:
# icons:
#   postgres: "󰆼"
#   redis: "󰟆"
#   mysql: "󰆚"
#   nginx: "󱏿"
#   node: "󰎙"
#   python: "󰌠"
#   default: "󰡨"

# Custom Commands:
# User-defined shell commands shown in the Action Menu ('x' key).
# Commands execute in the terminal with template variables:
#   Containers: {{ .ID }}, {{ .Name }}, {{ .Image }}
#   Images:     {{ .ID }}, {{ .Repository }}, {{ .Tag }}
#   Volumes:    {{ .Name }}
#   Networks:   {{ .ID }}, {{ .Name }}
#
# Options:
#   pause: true - Waits for user to press Enter before returning to docktop.
#                 Recommended for informational commands (e.g., inspect, history, top).
#
# Examples:
# customCommands:
#   containers:
#     - name: "Bash Shell"
#       command: "docker exec -it {{ .ID }} bash"
#     - name: "Inspect Container JSON"
#       command: "docker inspect {{ .ID }}"
#       pause: true
#     - name: "Top Processes"
#       command: "docker top {{ .ID }}"
#       pause: true
#   images:
#     - name: "Image History"
#       command: "docker history {{ .ID }}"
#       pause: true
#     - name: "Inspect Image"
#       command: "docker image inspect {{ .ID }}"
#       pause: true
#   volumes:
#     - name: "Browse Files (Busybox)"
#       command: "docker run --rm -it -v {{ .Name }}:/v busybox ls -la /v"
#       pause: true
#   networks:
#     - name: "Inspect Network"
#       command: "docker network inspect {{ .Name }}"
#       pause: true
`
	return os.WriteFile(path, []byte(template), 0o644)
}
