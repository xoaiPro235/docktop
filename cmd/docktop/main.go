package main

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/xoaiPro235/docktop/internal/config"
	"github.com/xoaiPro235/docktop/internal/docker"
	"github.com/xoaiPro235/docktop/internal/tui"
	"github.com/xoaiPro235/docktop/internal/tui/theme"
)

func main() {
	opts := parseFlags()

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Notice: using default configuration (%v)\n", err)
	}
	if cfg != nil && cfg.Theme != "" {
		theme.SetThemeByName(cfg.Theme)
	}

	// Precedence: CLI flag (-H/--host) > config file (host) > DOCKER_HOST env / default socket
	dockerHost := strings.TrimSpace(opts.host)
	if dockerHost == "" && cfg != nil {
		dockerHost = strings.TrimSpace(cfg.Host)
	}

	dockerCli, err := docker.NewClient(dockerHost)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to Docker Engine: %v\n", err)
		if dockerHost != "" {
			fmt.Fprintf(os.Stderr, "Target host: %s\n", dockerHost)
		}
		fmt.Fprintf(os.Stderr, "Hint: Make sure Docker daemon is running and reachable (check socket permissions or SSH connection).\n")
		os.Exit(1)
	}

	app := tui.NewApp(dockerCli, cfg)

	p := tea.NewProgram(app)
	_, runErr := p.Run()
	_ = dockerCli.Close()

	if runErr != nil {
		fmt.Fprintf(os.Stderr, "Error running docktop: %v\n", runErr)
		os.Exit(1)
	}
}
