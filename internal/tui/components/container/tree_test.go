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

	roots := BuildTree(containers, nil)
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

func Test_resolveComposeProject(t *testing.T) {
	tempDir := t.TempDir()
	composePath := filepath.Join(tempDir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte("services: {}"), 0o644); err != nil {
		t.Fatalf("Failed to write compose file: %v", err)
	}

	node := &TreeNode{
		Type:        NodeContainer,
		Name:        "web-1",
		ProjectName: "myproject",
		ConfigFiles: composePath,
		WorkingDir:  tempDir,
	}
	proj, svc, dir, file := resolveComposeProject(node)
	if proj != "myproject" {
		t.Errorf("got proj = %q, want myproject", proj)
	}
	if dir != tempDir || file != composePath {
		t.Errorf("got dir=%q, file=%q, want dir=%q, file=%q", dir, file, tempDir, composePath)
	}
	_ = svc
}

func TestFindComposeFile(t *testing.T) {
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

	childDir := t.TempDir()
	childFile := filepath.Join(childDir, "compose.yml")
	if err := os.WriteFile(childFile, []byte("services: {}"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		node *TreeNode
		want string
	}{
		{
			name: "uses first existing path in config file labels",
			node: &TreeNode{
				ConfigFiles: filepath.Join(rootDir, "missing.yaml") + ", " + rootFile,
			},
			want: rootFile,
		},
		{
			name: "falls back to compose file in root working directory",
			node: &TreeNode{WorkingDir: workingDir},
			want: workingFile,
		},
		{
			name: "finds compose file in nested container working directory",
			node: &TreeNode{
				Children: []*TreeNode{{
					Children: []*TreeNode{{
						Container: &docker.ContainerInfo{WorkingDir: childDir},
					}},
				}},
			},
			want: childFile,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := findComposeFile(tt.node); got != tt.want {
				t.Errorf("findComposeFile() = %q, want %q", got, tt.want)
			}
		})
	}
}
