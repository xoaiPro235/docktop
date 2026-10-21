package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/xoaiPro235/docktop/internal/config"
	"github.com/xoaiPro235/docktop/internal/docker"
	"github.com/xoaiPro235/docktop/internal/tui"
	"github.com/xoaiPro235/docktop/internal/tui/theme"
)

// Injected by GoReleaser ldflags
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Parse CLI flags
	var showVersion bool
	flag.BoolVar(&showVersion, "v", false, "Show version and exit")
	flag.BoolVar(&showVersion, "version", false, "Show version and exit")

	var cliHost string
	flag.StringVar(&cliHost, "H", "", "Docker daemon host to connect to (e.g. unix:///var/run/docker.sock, ssh://user@host)")
	flag.StringVar(&cliHost, "host", "", "Docker daemon host to connect to (alias for -H)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "docktop - A fast, modern terminal UI for Docker\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n  docktop [flags]\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if showVersion {
		fmt.Printf("docktop version %s (commit: %s, built at: %s)\n", version, commit, date)
		os.Exit(0)
	}

	// 1. Load configuration (~/.config/docktop/config.yaml)
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Notice: using default configuration (%v)\n", err)
	}
	if cfg != nil && cfg.Theme != "" {
		theme.SetThemeByName(cfg.Theme)
	}

	// 2. Resolve Docker host endpoint
	// Precedence: CLI flag (-H/--host) > config file (host) > DOCKER_HOST env / default socket
	dockerHost := strings.TrimSpace(cliHost)
	if dockerHost == "" && cfg != nil {
		dockerHost = strings.TrimSpace(cfg.Host)
	}

	// 3. Connect to Docker Engine
	dockerCli, err := docker.NewClient(dockerHost)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to Docker Engine: %v\n", err)
		if dockerHost != "" {
			fmt.Fprintf(os.Stderr, "Target host: %s\n", dockerHost)
		}
		fmt.Fprintf(os.Stderr, "Hint: Make sure Docker daemon is running and reachable (check socket permissions or SSH connection).\n")
		os.Exit(1)
	}
	defer dockerCli.Close()

	// 4. Initialize Bubble Tea Root Model
	app := tui.NewApp(dockerCli)

	// 5. Run application
	p := tea.NewProgram(app)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running docktop: %v\n", err)
		os.Exit(1)
	}
}
