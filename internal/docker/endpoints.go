package docker

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// resolveDefaultHost performs automatic 2-tier discovery to locate an active
// Docker daemon endpoint when neither an explicit CLI/config host nor DOCKER_HOST is provided.
//
// Resolution order:
//  1. Stage 1: Active Docker context (checks $DOCKER_CONTEXT or ~/.docker/config.json).
//  2. Stage 2: OS-specific local socket/pipe probing (OrbStack, Colima, WSL2 pipes, Rootless).
//
// Returns an empty string if no context or local socket can be detected, allowing the Moby client
// to fall back to its internal compile-time default.)
func resolveDefaultHost() string {
	// 1. Try resolving from active Docker Context
	if host := resolveDockerContextHost(); host != "" {
		return host
	}

	// 2. Fallback to probing OS-specific sockets/pipes
	return probeLocalSocket()
}

// getDockerConfigDir returns the directory containing Docker CLI configurations.
// Respects the $DOCKER_CONFIG environment variable if set, otherwise defaults to ~/.docker.
func getDockerConfigDir() string {
	if dir := os.Getenv("DOCKER_CONFIG"); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".docker")
}

// resolveDockerContextHost inspects the active Docker CLI context and extracts its daemon host endpoint.
//
// Context discovery precedence:
//  1. $DOCKER_CONTEXT environment variable (highest context priority)
//  2. "currentContext" field in ~/.docker/config.json
//
// Context metadata is stored under ~/.docker/contexts/meta/<sha256(context_name)>/meta.json
// matching the storage format used by the Docker CLI (v19.03+).
// Returns an empty string if the context is unset, set to "default", or unreadable.
func resolveDockerContextHost() string {
	configDir := getDockerConfigDir()
	if configDir == "" {
		return ""
	}

	contextName := strings.TrimSpace(os.Getenv("DOCKER_CONTEXT"))

	if contextName == "" {
		configFile := filepath.Join(configDir, "config.json")
		data, err := os.ReadFile(configFile)
		if err == nil {
			var cfg struct {
				CurrentContext string `json:"currentContext"`
			}
			if err := json.Unmarshal(data, &cfg); err == nil {
				contextName = strings.TrimSpace(cfg.CurrentContext)
			}
		}
	}

	if contextName == "" || contextName == "default" {
		return ""
	}

	// Context metadata is stored under contexts/meta/<sha256(name)>/meta.json
	h := sha256.Sum256([]byte(contextName))
	contextHash := hex.EncodeToString(h[:])

	metaFile := filepath.Join(configDir, "contexts", "meta", contextHash, "meta.json")
	metaData, err := os.ReadFile(metaFile)
	if err != nil {
		return ""
	}

	var meta struct {
		Endpoints struct {
			Docker struct {
				Host string `json:"Host"`
			} `json:"docker"`
		} `json:"Endpoints"`
	}
	if err := json.Unmarshal(metaData, &meta); err != nil {
		return ""
	}

	return meta.Endpoints.Docker.Host
}

// probeLocalSocket inspects known OS-specific socket paths and named pipes on disk
// to locate an active Docker engine without requiring manual configuration.
//
// Platform support:
//   - Windows: Checks modern Docker Desktop WSL2 pipe (dockerDesktopLinuxEngine),
//     then legacy Docker CE engine pipe (docker_engine).
//   - macOS: Checks standard system socket (/var/run/docker.sock), then rootless
//     user sockets in $HOME (~/.docker/run, ~/.orbstack/run, ~/.colima/default).
//   - Linux: Checks standard system socket (/var/run/docker.sock), then Rootless Docker
//     socket via $XDG_RUNTIME_DIR/docker.sock.
func probeLocalSocket() string {
	switch runtime.GOOS {
	case "windows":
		if _, err := os.Stat(`\\.\pipe\dockerDesktopLinuxEngine`); err == nil {
			return "npipe:////./pipe/dockerDesktopLinuxEngine"
		}
		if _, err := os.Stat(`\\.\pipe\docker_engine`); err == nil {
			return "npipe:////./pipe/docker_engine"
		}

	case "darwin":
		if _, err := os.Stat("/var/run/docker.sock"); err == nil {
			return "unix:///var/run/docker.sock"
		}
		home, err := os.UserHomeDir()
		if err == nil {
			candidates := []string{
				filepath.Join(home, ".docker", "run", "docker.sock"),
				filepath.Join(home, ".orbstack", "run", "docker.sock"),
				filepath.Join(home, ".colima", "default", "docker.sock"),
			}
			for _, sock := range candidates {
				if _, err := os.Stat(sock); err == nil {
					return "unix://" + sock
				}
			}
		}

	case "linux":
		if _, err := os.Stat("/var/run/docker.sock"); err == nil {
			return "unix:///var/run/docker.sock"
		}
		if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
			sock := filepath.Join(xdg, "docker.sock")
			if _, err := os.Stat(sock); err == nil {
				return "unix://" + sock
			}
		}
	}

	return ""
}
