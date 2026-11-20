package container

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xoaiPro235/docktop/internal/docker"
)

func TestTreeNode_DisplayTitle(t *testing.T) {
	// Project node
	projNode := &TreeNode{
		Type:     NodeProject,
		Name:     "my-stack",
		Expanded: true,
	}
	if title := projNode.DisplayTitle(); !strings.Contains(title, "▼") || !strings.Contains(title, "my-stack") {
		t.Errorf("Project DisplayTitle() = %q", title)
	}

	// Collapsed standalone node
	standaloneNode := &TreeNode{
		Type:     NodeProject,
		Name:     StandaloneProjectName,
		Expanded: false,
		Children: []*TreeNode{{Type: NodeContainer, Name: "c1"}},
	}
	if title := standaloneNode.DisplayTitle(); !strings.Contains(title, "▶") || !strings.Contains(title, "Standalone (1)") {
		t.Errorf("Standalone DisplayTitle() = %q", title)
	}

	// Service node
	svcNode := &TreeNode{
		Type:     NodeService,
		Name:     "web",
		Expanded: true,
		Children: []*TreeNode{
			{Type: NodeContainer, Name: "c1"},
			{Type: NodeContainer, Name: "c2"},
		},
	}
	if title := svcNode.DisplayTitle(); !strings.Contains(title, "web (2)") {
		t.Errorf("Service DisplayTitle() = %q", title)
	}

	// Running Container node
	ctrRunning := &TreeNode{
		Type: NodeContainer,
		Name: "web-1",
		Container: &docker.ContainerInfo{
			State: docker.StateRunning,
		},
	}
	if title := ctrRunning.DisplayTitle(); !strings.Contains(title, "●") || !strings.Contains(title, "web-1") {
		t.Errorf("Running container DisplayTitle() = %q", title)
	}

	// Running Container node with Icon
	ctrRunningWithIcon := &TreeNode{
		Type: NodeContainer,
		Name: "web-1",
		Icon: "󰆼",
		Container: &docker.ContainerInfo{
			State: docker.StateRunning,
		},
	}
	if title := ctrRunningWithIcon.DisplayTitle(); !strings.Contains(title, "● 󰆼 web-1") {
		t.Errorf("Running container with icon DisplayTitle() = %q, want to contain '● 󰆼 web-1'", title)
	}

	// Stopped Container node with OneOff
	ctrStopped := &TreeNode{
		Type: NodeContainer,
		Name: "migrate-1",
		Container: &docker.ContainerInfo{
			State:  docker.StateExited,
			OneOff: true,
		},
	}
	if title := ctrStopped.DisplayTitle(); !strings.Contains(title, "○") || !strings.Contains(title, "[one-off]") {
		t.Errorf("Stopped container DisplayTitle() = %q", title)
	}
}

func TestMatchIcon(t *testing.T) {
	icons := map[string]string{
		"postgres":      "󰆼",
		"redis":         "󰟆",
		"mysql":         "󰆚",
		"node":          "󰎙",
		"mssql":         "",
		"mssql/server":  "",
		"my-custom-app": "🚀",
		"web_worker":    "⚙️",
		"default":       "󰡨",
	}

	tests := []struct {
		name          string
		imageName     string
		containerName string
		want          string
	}{
		{"exact repository match with tag", "postgres:16-alpine", "db-1", "󰆼"},
		{"registry and repo path", "docker.io/bitnami/redis:7.2", "cache", "󰟆"},
		{"private registry with port", "localhost:5000/company/redis:7.2", "cache", "󰟆"},
		{"digest stripped properly", "mysql:8.0@sha256:abcdef1234567890", "db", "󰆚"},
		{"token in repo name", "ghcr.io/org/my-node-service:v2", "api", "󰎙"},
		{"token in container name", "custom-internal-image:latest", "prod-mysql-slave", "󰆚"},
		{"delimiter-containing container name exact match (hyphen)", "custom-image:latest", "my-custom-app", "🚀"},
		{"delimiter-containing container name exact match (underscore)", "scratch:latest", "web_worker", "⚙️"},
		{"container name overrides image icon", "postgres:16-alpine", "my-custom-app", "🚀"},
		{"multi-part registry repo token match (mssql)", "mcr.microsoft.com/mssql/server:latest", "sql1", ""},
		{"multi-part registry exact repo path match", "mcr.microsoft.com/mssql/server:2022-CU14-ubuntu-22.04", "db", ""},
		{"token false-positive avoidance", "algo-service:latest", "trader", "󰡨"},
		{"fallback to default", "unknown-vendor/secret-app:1.0", "worker", "󰡨"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MatchIcon(tt.imageName, tt.containerName, icons); got != tt.want {
				t.Errorf("MatchIcon(%q, %q) = %q, want %q", tt.imageName, tt.containerName, got, tt.want)
			}
		})
	}

	// Empty map returns empty string
	if got := MatchIcon("postgres", "db", nil); got != "" {
		t.Errorf("MatchIcon with nil map = %q, want empty string", got)
	}
}

func Test_containerStateScore(t *testing.T) {
	tests := []struct {
		state docker.ContainerState
		want  int
	}{
		{docker.StateRunning, 1},
		{docker.StateRestarting, 2},
		{docker.StatePaused, 3},
		{docker.StateCreated, 4},
		{docker.StateExited, 5},
		{docker.StateDead, 5},
		{docker.ContainerState("unknown"), 6},
	}
	for _, tt := range tests {
		if got := containerStateScore(tt.state); got != tt.want {
			t.Errorf("containerStateScore(%q) = %d, want %d", string(tt.state), got, tt.want)
		}
	}
}

func Test_compareContainerNodes(t *testing.T) {
	// Running (score 1) vs Exited (score 5)
	n1 := &TreeNode{
		Name:      "c1",
		Container: &docker.ContainerInfo{State: docker.StateRunning},
	}
	n2 := &TreeNode{
		Name:      "c2",
		Container: &docker.ContainerInfo{State: docker.StateExited},
	}
	if cmp := compareContainerNodes(n1, n2); cmp >= 0 {
		t.Errorf("Running container should come before exited container, got %d", cmp)
	}

	// Same state, different container numbers
	n3 := &TreeNode{
		Name:      "web_2",
		Container: &docker.ContainerInfo{State: docker.StateRunning, ContainerNumber: 2},
	}
	n4 := &TreeNode{
		Name:      "web_10",
		Container: &docker.ContainerInfo{State: docker.StateRunning, ContainerNumber: 10},
	}
	if cmp := compareContainerNodes(n3, n4); cmp >= 0 {
		t.Errorf("ContainerNumber 2 should come before 10, got %d", cmp)
	}
}

func Test_isNodeExpanded(t *testing.T) {
	// nil stateMap defaults to true
	if !isNodeExpanded("any", nil) {
		t.Error("nil stateMap should default to true")
	}

	stateMap := map[string]bool{
		"proj:open":   true,
		"proj:closed": false,
	}
	if !isNodeExpanded("proj:open", stateMap) {
		t.Error("proj:open should be true")
	}
	if isNodeExpanded("proj:closed", stateMap) {
		t.Error("proj:closed should be false")
	}
	if !isNodeExpanded("proj:unknown", stateMap) {
		t.Error("unknown key should default to true")
	}
}

func Test_findOrCreateChild(t *testing.T) {
	parent := &TreeNode{
		Name: "myproj",
	}
	child1 := findOrCreateChild(parent, "db", NodeService, nil)
	if child1 == nil || child1.Name != "db" {
		t.Fatal("Failed to create child")
	}

	// Calling again returns existing child
	child2 := findOrCreateChild(parent, "db", NodeService, nil)
	if child1 != child2 {
		t.Error("Expected findOrCreateChild to return existing child")
	}
	if len(parent.Children) != 1 {
		t.Errorf("Expected 1 child, got %d", len(parent.Children))
	}
}

func TestBuildTreeAndFlatten(t *testing.T) {
	containers := []docker.ContainerInfo{
		// Compose container 1
		{
			ID:          "c1",
			Name:        "myapp-web-1",
			IsCompose:   true,
			ProjectName: "myapp",
			Service:     "web",
			State:       docker.StateRunning,
		},
		// Compose container 2
		{
			ID:          "c2",
			Name:        "myapp-db-1",
			IsCompose:   true,
			ProjectName: "myapp",
			Service:     "db",
			State:       docker.StateRunning,
		},
		// Standalone container
		{
			ID:        "c3",
			Name:      "redis-standalone",
			IsCompose: false,
			State:     docker.StateRunning,
		},
	}

	roots := BuildTree(containers, nil, nil)
	if len(roots) != 2 { // "myapp" and standalone
		t.Fatalf("BuildTree roots count = %d, want 2", len(roots))
	}

	// Verify Flattening
	flatItems := Flatten(roots, "")
	if len(flatItems) == 0 {
		t.Fatal("Flatten returned empty list")
	}

	foundWeb := false
	foundStandalone := false
	for _, item := range flatItems {
		if strings.Contains(item.Node.Name, "myapp-web-1") {
			foundWeb = true
		}
		if item.Node.Name == "redis-standalone" {
			foundStandalone = true
		}
	}
	if !foundWeb || !foundStandalone {
		t.Errorf("Flatten missing expected containers: foundWeb=%v, foundStandalone=%v", foundWeb, foundStandalone)
	}
}

func TestTreeNode_ComposeOpts(t *testing.T) {
	tempDir := t.TempDir()
	composePath := filepath.Join(tempDir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte("services: {}"), 0o644); err != nil {
		t.Fatalf("Failed to write compose file: %v", err)
	}

	tests := []struct {
		name        string
		node        *TreeNode
		wantProject string
		wantService string
	}{
		{
			name: "project",
			node: &TreeNode{
				Type:        NodeProject,
				Name:        "myproject",
				ConfigFiles: composePath,
				WorkingDir:  tempDir,
			},
			wantProject: "myproject",
		},
		{
			name: "service",
			node: &TreeNode{
				Type:        NodeService,
				Name:        "web",
				ProjectName: "myproject",
				ConfigFiles: composePath,
				WorkingDir:  tempDir,
			},
			wantProject: "myproject",
			wantService: "web",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := tt.node.ComposeOpts()
			if opts.ProjectName != tt.wantProject {
				t.Errorf("ProjectName = %q, want %q", opts.ProjectName, tt.wantProject)
			}
			if opts.Service != tt.wantService {
				t.Errorf("Service = %q, want %q", opts.Service, tt.wantService)
			}
			if opts.WorkingDir != tempDir || opts.ConfigFile != composePath {
				t.Errorf("ComposeOpts() = {WorkingDir: %q, ConfigFile: %q}, want {WorkingDir: %q, ConfigFile: %q}",
					opts.WorkingDir, opts.ConfigFile, tempDir, composePath)
			}
		})
	}
}

func TestTreeNode_ComposeOptsFindsComposeFile(t *testing.T) {
	rootDir := t.TempDir()
	rootFile := filepath.Join(rootDir, "compose.yaml")
	if err := os.WriteFile(rootFile, []byte("services: {}"), 0o644); err != nil {
		t.Fatal(err)
	}

	workingDir := t.TempDir()
	workingFile := filepath.Join(workingDir, "docker-compose.yml")
	if err := os.WriteFile(workingFile, []byte("services: {}"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name        string
		node        *TreeNode
		wantFile    string
		wantWorkDir string
	}{
		{
			name: "uses first existing config file from labels",
			node: &TreeNode{
				ConfigFiles: filepath.Join(rootDir, "missing.yaml") + ", " + rootFile,
			},
			wantFile:    rootFile,
			wantWorkDir: rootDir,
		},
		{
			name:        "finds a compose file in the working directory",
			node:        &TreeNode{WorkingDir: workingDir},
			wantFile:    workingFile,
			wantWorkDir: workingDir,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := tt.node.ComposeOpts()
			if opts.ConfigFile != tt.wantFile || opts.WorkingDir != tt.wantWorkDir {
				t.Errorf("ComposeOpts() = {ConfigFile: %q, WorkingDir: %q}, want {ConfigFile: %q, WorkingDir: %q}",
					opts.ConfigFile, opts.WorkingDir, tt.wantFile, tt.wantWorkDir)
			}
		})
	}
}
