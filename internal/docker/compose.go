package docker

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ComposeCommandType represents the variant of Docker Compose available on the host.
type ComposeCommandType string

const (
	ComposeV2   ComposeCommandType = "docker compose"
	ComposeV1   ComposeCommandType = "docker-compose"
	ComposeNone ComposeCommandType = "none"
)

var (
	ErrComposeNotAvailable = errors.New("Docker Compose is not available on this system")

	// DefaultComposeFileNames contains standard filenames for Docker Compose definitions in order of precedence.
	DefaultComposeFileNames = []string{
		"compose.yaml",
		"compose.yml",
		"docker-compose.yaml",
		"docker-compose.yml",
	}
)

// ResolveExistingFile checks if a path exists, handling Windows/WSL2 path conversions if applicable.
func ResolveExistingFile(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if _, err := os.Stat(path); err == nil {
		return path
	}

	// On Windows, paths from containers running in WSL2 often look like /mnt/c/... or //wsl$/...
	if runtime.GOOS == "windows" {
		// Pattern 1: Normalize forward-slash UNC paths (e.g. //wsl$/... or //wsl.localhost/...)
		if strings.HasPrefix(path, "//") || strings.HasPrefix(path, `\\`) {
			winUNC := filepath.FromSlash(path)
			if _, err := os.Stat(winUNC); err == nil {
				return winUNC
			}
		}

		// Pattern 2: Convert /mnt/c/... to C:\...
		if strings.HasPrefix(path, "/mnt/") && len(path) > 6 && path[6] == '/' {
			ch := path[5]
			if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
				driveLetter := strings.ToUpper(string(ch))
				winPath := driveLetter + `:\` + filepath.FromSlash(path[7:])
				if _, err := os.Stat(winPath); err == nil {
					return winPath
				}
			}
		}
	}
	return ""
}

// FindComposeFile attempts to locate an existing compose file from comma-separated configFiles or workingDir.
func FindComposeFile(configFiles, workingDir string) string {
	if configFiles != "" {
		for f := range strings.SplitSeq(configFiles, ",") {
			if resolved := ResolveExistingFile(f); resolved != "" {
				return resolved
			}
		}
	}

	workingDir = strings.TrimSpace(workingDir)
	if workingDir != "" {
		for _, name := range DefaultComposeFileNames {
			candidate := filepath.Join(workingDir, name)
			if resolved := ResolveExistingFile(candidate); resolved != "" {
				return resolved
			}
		}
	}

	return ""
}

// ComposeOpts specifies parameters to target a Docker Compose project or service.
type ComposeOpts struct {
	ProjectName string
	Service     string
	WorkingDir  string
	ConfigFile  string
}

// LocalComposeProject holds metadata discovered from a local compose file.
type LocalComposeProject struct {
	ProjectName string
	WorkingDir  string
	ConfigFile  string
	Services    []string
}

// NormalizeProjectName converts a directory or project name to a valid compose project name
// by lowercasing and stripping characters that are not [a-z0-9_-].
func NormalizeProjectName(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// FindComposeFileInAncestors starts at dir and walks up parent directories until a compose file is found.
// It returns the path to the compose file and the directory containing it.
// If dir is empty, it uses the current working directory.
func FindComposeFileInAncestors(dir string) (filePath string, foundDir string) {
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return "", ""
		}
	}
	absDir, err := filepath.Abs(dir)
	if err != nil {
		absDir = dir
	}

	curr := absDir
	for {
		if f := FindComposeFile("", curr); f != "" {
			return f, curr
		}
		parent := filepath.Dir(curr)
		if parent == curr {
			break
		}
		curr = parent
	}
	return "", ""
}

// DetectLocalCompose scans the given directory and its parent directories
// for compose files and parses service names.
// If dir is empty, it uses the current working directory.
func DetectLocalCompose(dir string) (*LocalComposeProject, error) {
	foundPath, projectDir := FindComposeFileInAncestors(dir)
	if foundPath == "" {
		return nil, nil
	}

	data, err := os.ReadFile(foundPath)
	if err != nil {
		return nil, err
	}

	var parsed struct {
		Name     string         `yaml:"name"`
		Services map[string]any `yaml:"services"`
	}

	if err := yaml.Unmarshal(data, &parsed); err != nil {
		return nil, err
	}

	projectName := resolveProjectName(projectDir, parsed.Name)

	var services []string
	for svcName := range parsed.Services {
		services = append(services, svcName)
	}
	slices.Sort(services)

	return &LocalComposeProject{
		ProjectName: projectName,
		WorkingDir:  projectDir,
		ConfigFile:  foundPath,
		Services:    services,
	}, nil
}

// resolveProjectName determines the compose project name following Docker Compose's official precedence:
// 1. COMPOSE_PROJECT_NAME environment variable
// 2. COMPOSE_PROJECT_NAME from .env file in projectDir
// 3. Top-level "name:" attribute in compose file
// 4. Directory base name
func resolveProjectName(projectDir, yamlName string) string {
	if val := strings.TrimSpace(os.Getenv("COMPOSE_PROJECT_NAME")); val != "" {
		return NormalizeProjectName(val)
	}
	if val := parseDotEnvProjectName(projectDir); val != "" {
		return NormalizeProjectName(val)
	}
	if yamlName != "" {
		return NormalizeProjectName(yamlName)
	}
	return NormalizeProjectName(filepath.Base(projectDir))
}

// parseDotEnvProjectName extracts COMPOSE_PROJECT_NAME from a .env file if present.
func parseDotEnvProjectName(projectDir string) string {
	envData, err := os.ReadFile(filepath.Join(projectDir, ".env"))
	if err != nil {
		return ""
	}
	for line := range strings.SplitSeq(string(envData), "\n") {
		line = strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(line, "COMPOSE_PROJECT_NAME="); ok {
			val := after
			val = strings.Trim(val, `"' `)
			if val != "" {
				return val
			}
		}
	}
	return ""
}

var detectComposeFunc = defaultDetectCompose

func defaultDetectCompose() (binary string, subcmd []string, cmdType ComposeCommandType) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 1. Try modern "docker compose version"
	if err := exec.CommandContext(ctx, "docker", "compose", "version").Run(); err == nil {
		return "docker", []string{"compose"}, ComposeV2
	}
	// 2. Fallback to legacy "docker-compose version"
	if ctx.Err() == nil {
		ctx2, cancel2 := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel2()
		if err := exec.CommandContext(ctx2, "docker-compose", "version").Run(); err == nil {
			return "docker-compose", nil, ComposeV1
		}
	}
	return "", nil, ComposeNone
}

func (c *Client) buildComposeCmd(ctx context.Context, opts ComposeOpts, subArgs ...string) (*exec.Cmd, error) {
	bin, prefix, cmdType := detectComposeFunc()
	if cmdType == ComposeNone {
		return nil, ErrComposeNotAvailable
	}

	var args []string
	if c.host != "" && bin == "docker" {
		args = append(args, "-H", c.host)
	}
	args = append(args, prefix...)
	if cmdType == ComposeV2 && len(subArgs) > 0 && subArgs[0] == "logs" {
		args = append(args, "--ansi", "always")
	}

	if opts.ProjectName != "" {
		args = append(args, "-p", opts.ProjectName)
	}
	if opts.ConfigFile != "" {
		args = append(args, "-f", opts.ConfigFile)
	}
	if opts.WorkingDir != "" && cmdType == ComposeV2 {
		args = append(args, "--project-directory", opts.WorkingDir)
	}

	args = append(args, subArgs...)

	if opts.Service != "" {
		args = append(args, opts.Service)
	}

	cmd := exec.CommandContext(ctx, bin, args...)
	if opts.WorkingDir != "" {
		cmd.Dir = opts.WorkingDir
	}
	return cmd, nil
}

// ComposeUpCmd returns a command to start services in detached mode.
func (c *Client) ComposeUpCmd(ctx context.Context, opts ComposeOpts) (*exec.Cmd, error) {
	return c.buildComposeCmd(ctx, opts, "up", "-d")
}

// ComposeBuildCmd returns a command to build images and start services in detached mode.
func (c *Client) ComposeBuildCmd(ctx context.Context, opts ComposeOpts) (*exec.Cmd, error) {
	return c.buildComposeCmd(ctx, opts, "up", "-d", "--build")
}

// ComposeDownCmd returns a command to stop and remove containers, networks created by compose.
func (c *Client) ComposeDownCmd(ctx context.Context, opts ComposeOpts) (*exec.Cmd, error) {
	return c.buildComposeCmd(ctx, opts, "down")
}

// ComposeRestartCmd returns a command to restart compose services.
func (c *Client) ComposeRestartCmd(ctx context.Context, opts ComposeOpts) (*exec.Cmd, error) {
	return c.buildComposeCmd(ctx, opts, "restart")
}

// ComposeStopCmd returns a command to stop compose services.
func (c *Client) ComposeStopCmd(ctx context.Context, opts ComposeOpts) (*exec.Cmd, error) {
	return c.buildComposeCmd(ctx, opts, "stop")
}

// ComposeStartCmd returns a command to start stopped compose services.
func (c *Client) ComposeStartCmd(ctx context.Context, opts ComposeOpts) (*exec.Cmd, error) {
	return c.buildComposeCmd(ctx, opts, "start")
}

// ComposeRmCmd returns a command to remove stopped service containers.
func (c *Client) ComposeRmCmd(ctx context.Context, opts ComposeOpts) (*exec.Cmd, error) {
	return c.buildComposeCmd(ctx, opts, "rm", "-f", "-s")
}

// ComposeLogsCmd returns a command to stream compose logs.
func (c *Client) ComposeLogsCmd(ctx context.Context, opts ComposeOpts, tail string) (*exec.Cmd, error) {
	if tail == "" {
		tail = "50"
	}
	return c.buildComposeCmd(ctx, opts, "logs", "-f", "--tail", tail)
}
