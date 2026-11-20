package container

import (
	"cmp"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/xoaiPro235/docktop/internal/docker"
)

type NodeType int

const (
	NodeProject NodeType = iota
	NodeService
	NodeContainer
)

const StandaloneProjectName = "__standalone__"

type TreeNode struct {
	Type        NodeType
	Name        string
	ProjectName string
	Key         string
	Icon        string
	Container   *docker.ContainerInfo
	ConfigFiles string
	WorkingDir  string
	Expanded    bool
	Children    []*TreeNode
}

func (n *TreeNode) CountContainers() int {
	if n == nil {
		return 0
	}

	if n.Type == NodeContainer {
		return 1
	}
	count := 0
	for _, child := range n.Children {
		count += child.CountContainers()
	}
	return count
}

// HasRunningContainers returns true if at least one container within this node is currently running.
func (n *TreeNode) HasRunningContainers() bool {
	if n == nil {
		return false
	}
	if n.Type == NodeContainer {
		return n.Container != nil && n.Container.State == docker.StateRunning
	}
	for _, child := range n.Children {
		if child.HasRunningContainers() {
			return true
		}
	}
	return false
}

func (n *TreeNode) DisplayTitle() string {
	if n == nil {
		return ""
	}
	arrow := "▼"
	if !n.Expanded {
		arrow = "▶"
	}

	switch n.Type {
	case NodeProject:
		if n.Name == StandaloneProjectName {
			return fmt.Sprintf("%s Standalone (%d)", arrow, n.CountContainers())
		}
		return fmt.Sprintf("%s %s (%d)", arrow, n.Name, n.CountContainers())
	case NodeService:
		return fmt.Sprintf("%s %s (%d)", arrow, n.Name, n.CountContainers())
	case NodeContainer:
		statusDot := "●"
		if n.Container != nil && n.Container.State != docker.StateRunning {
			statusDot = "○"
		}
		iconPrefix := ""
		if n.Icon != "" {
			iconPrefix = n.Icon + " "
		}
		title := fmt.Sprintf("%s %s%s", statusDot, iconPrefix, n.Name)
		if n.Container != nil && n.Container.OneOff {
			title += " [one-off]"
		}
		return title
	default:
		return n.Name
	}
}

// ComposeOpts extracts Docker Compose parameters directly from a TreeNode.
func (n *TreeNode) ComposeOpts() docker.ComposeOpts {
	if n == nil {
		return docker.ComposeOpts{}
	}
	var proj, svc string
	if n.Type == NodeProject && n.Name != StandaloneProjectName {
		proj = n.Name
		if n.ProjectName != "" {
			proj = n.ProjectName
		}
	} else if n.Type == NodeService {
		proj = n.ProjectName
		svc = n.Name
	}

	configFile := docker.FindComposeFile(n.ConfigFiles, n.WorkingDir)
	workDir := n.WorkingDir
	if configFile != "" {
		workDir = filepath.Dir(configFile)
	}

	return docker.ComposeOpts{
		ProjectName: proj,
		Service:     svc,
		WorkingDir:  workDir,
		ConfigFile:  configFile,
	}
}

// IsCompose returns true if this node represents a Docker Compose project or service.
func (n *TreeNode) IsCompose() bool {
	return n != nil && n.Name != StandaloneProjectName && (n.Type == NodeProject || n.Type == NodeService)
}

type FlatItem struct {
	Node   *TreeNode
	Prefix string
}

func containerStateScore(state docker.ContainerState) int {
	switch state {
	case docker.StateRunning:
		return 1
	case docker.StateRestarting:
		return 2
	case docker.StatePaused:
		return 3
	case docker.StateCreated:
		return 4
	case docker.StateExited, docker.StateDead:
		return 5
	default:
		return 6
	}
}

func compareContainerNodes(a, b *TreeNode) int {
	scoreA := 99
	scoreB := 99
	if a.Container != nil {
		scoreA = containerStateScore(a.Container.State)
	}

	if b.Container != nil {
		scoreB = containerStateScore(b.Container.State)
	}

	if c := cmp.Compare(scoreA, scoreB); c != 0 {
		return c
	}

	// Sort scaled compose containers numerically by ContainerNumber (e.g. #2 before #10)
	if a.Container != nil && b.Container != nil {
		if a.Container.ContainerNumber > 0 && b.Container.ContainerNumber > 0 && a.Container.ContainerNumber != b.Container.ContainerNumber {
			return cmp.Compare(a.Container.ContainerNumber, b.Container.ContainerNumber)
		}
	}

	return cmp.Compare(a.Name, b.Name)
}

func isNodeExpanded(key string, stateMap map[string]bool) bool {
	if stateMap == nil {
		return true
	}
	if exp, ok := stateMap[key]; ok {
		return exp
	}
	return true
}

func findOrCreateChild(parent *TreeNode, name string, nodeType NodeType, stateMap map[string]bool) *TreeNode {
	if parent == nil {
		return nil
	}
	for _, child := range parent.Children {
		if child.Name == name {
			return child
		}
	}
	key := fmt.Sprintf("svc:%s/%s", parent.Name, name)
	newNode := &TreeNode{
		Type:     nodeType,
		Name:     name,
		Key:      key,
		Expanded: isNodeExpanded(key, stateMap),
	}
	parent.Children = append(parent.Children, newNode)
	return newNode
}

// MatchIcon finds an appropriate Nerd Font icon for a container given its image and container name.
// It prioritizes exact container name match, exact repository match (full path then base repo),
// token keyword match on image repo and container name, and falls back to icons["default"].
func MatchIcon(imageName, containerName string, icons map[string]string) string {
	if len(icons) == 0 {
		return ""
	}

	// 1. Direct exact match on normalized container name (custom per-container icon wins over image)
	cName := strings.ToLower(strings.TrimSpace(containerName))
	if icon, ok := icons[cName]; ok && cName != "default" && icon != "" {
		return icon
	}

	img := strings.ToLower(strings.TrimSpace(imageName))
	// Strip digest (@sha256:...)
	if idx := strings.Index(img, "@"); idx != -1 {
		img = img[:idx]
	}
	// Strip tag (:latest) after the last slash (retains registry:port intact)
	lastSlash := strings.LastIndex(img, "/")
	if lastSlash != -1 {
		if idx := strings.Index(img[lastSlash+1:], ":"); idx != -1 {
			img = img[:lastSlash+1+idx]
		}
	} else if idx := strings.Index(img, ":"); idx != -1 {
		img = img[:idx]
	}

	// Extract repository path by stripping registry host if present
	firstSlash := strings.Index(img, "/")
	repoPath := img
	if firstSlash != -1 {
		domain := img[:firstSlash]
		if strings.Contains(domain, ".") || strings.Contains(domain, ":") || domain == "localhost" {
			repoPath = img[firstSlash+1:]
		}
	}

	baseRepo := repoPath
	if idx := strings.LastIndex(repoPath, "/"); idx != -1 {
		baseRepo = repoPath[idx+1:]
	}

	// 2. Direct exact match on full repository path (e.g. "mssql/server", "bitnami/redis")
	if icon, ok := icons[repoPath]; ok && repoPath != "default" && icon != "" {
		return icon
	}

	// 3. Direct exact match on base repository (e.g. "postgres", "redis", "nginx")
	if baseRepo != repoPath {
		if icon, ok := icons[baseRepo]; ok && baseRepo != "default" && icon != "" {
			return icon
		}
	}

	// 4. Token match on repository path in reverse order (e.g. prefers product over vendor)
	tokens := splitNameTokens(repoPath)
	for _, token := range slices.Backward(tokens) {

		if icon, ok := icons[token]; ok && token != "default" && icon != "" {
			return icon
		}
	}

	// 5. Token match on container name (e.g. "my_redis_cache" -> ["my", "redis", "cache"])
	for _, token := range splitNameTokens(cName) {
		if icon, ok := icons[token]; ok && token != "default" && icon != "" {
			return icon
		}
	}

	// 6. Fallback to default icon
	if def, ok := icons["default"]; ok {
		return def
	}

	return ""
}

func splitNameTokens(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return r == '-' || r == '_' || r == '.' || r == '/' || r == ':'
	})
}

var detectLocalComposeFunc = docker.DetectLocalCompose

func BuildTree(containers []docker.ContainerInfo, stateMap map[string]bool, icons map[string]string) []*TreeNode {
	var roots []*TreeNode
	var standaloneNode *TreeNode

	for i := range containers {
		c := &containers[i]

		if c.IsCompose && c.ProjectName != "" {
			cleanFile := docker.FindComposeFile(c.ConfigFiles, c.WorkingDir)
			workDir := c.WorkingDir
			if cleanFile != "" {
				workDir = filepath.Dir(cleanFile)
			} else {
				cleanFile = c.ConfigFiles
			}

			var projNode *TreeNode
			for _, r := range roots {
				if r.Name == c.ProjectName {
					projNode = r
					break
				}
			}
			if projNode == nil {
				key := "proj:" + c.ProjectName
				projNode = &TreeNode{
					Type:        NodeProject,
					Name:        c.ProjectName,
					ProjectName: c.ProjectName,
					Key:         key,
					ConfigFiles: cleanFile,
					WorkingDir:  workDir,
					Expanded:    isNodeExpanded(key, stateMap),
				}
				roots = append(roots, projNode)
			} else {
				if projNode.ConfigFiles == "" && cleanFile != "" {
					projNode.ConfigFiles = cleanFile
					projNode.WorkingDir = workDir
				}
			}

			svcName := c.Service
			if svcName == "" {
				svcName = "other"
			}
			svcNode := findOrCreateChild(projNode, svcName, NodeService, stateMap)
			if svcNode.ProjectName == "" {
				svcNode.ProjectName = c.ProjectName
			}
			if svcNode.ConfigFiles == "" && cleanFile != "" {
				svcNode.ConfigFiles = cleanFile
				svcNode.WorkingDir = workDir
			}

			ctrKey := c.ID
			if ctrKey == "" {
				ctrKey = c.Name
			}
			ctrIcon := MatchIcon(c.Image, c.Name, icons)
			svcNode.Children = append(svcNode.Children, &TreeNode{
				Type:        NodeContainer,
				Name:        c.Name,
				Icon:        ctrIcon,
				ProjectName: c.ProjectName,
				Key:         "ctr:" + ctrKey,
				Container:   c,
			})
		} else {
			if standaloneNode == nil {
				key := "standalone"
				standaloneNode = &TreeNode{
					Type:     NodeProject,
					Name:     StandaloneProjectName,
					Key:      key,
					Expanded: isNodeExpanded(key, stateMap),
				}
			}
			ctrKey := c.ID
			if ctrKey == "" {
				ctrKey = c.Name
			}
			ctrIcon := MatchIcon(c.Image, c.Name, icons)
			standaloneNode.Children = append(standaloneNode.Children, &TreeNode{
				Type:      NodeContainer,
				Name:      c.Name,
				Icon:      ctrIcon,
				Key:       "ctr:" + ctrKey,
				Container: c,
			})
		}
	}

	// Detect local compose project in current working directory (if any)
	if detectLocalComposeFunc != nil {
		if localProj, _ := detectLocalComposeFunc(""); localProj != nil && localProj.ProjectName != "" {
			var existingProj *TreeNode
			for _, r := range roots {
				if strings.EqualFold(r.Name, localProj.ProjectName) ||
					(r.WorkingDir != "" && localProj.WorkingDir != "" && filepath.Clean(r.WorkingDir) == filepath.Clean(localProj.WorkingDir)) ||
					(r.ConfigFiles != "" && localProj.ConfigFile != "" && filepath.Clean(r.ConfigFiles) == filepath.Clean(localProj.ConfigFile)) {
					existingProj = r
					break
				}
			}

			if existingProj == nil {
				key := "proj:" + localProj.ProjectName
				existingProj = &TreeNode{
					Type:        NodeProject,
					Name:        localProj.ProjectName,
					ProjectName: localProj.ProjectName,
					Key:         key,
					ConfigFiles: localProj.ConfigFile,
					WorkingDir:  localProj.WorkingDir,
					Expanded:    isNodeExpanded(key, stateMap),
				}
				roots = append(roots, existingProj)
			} else {
				if existingProj.ConfigFiles == "" {
					existingProj.ConfigFiles = localProj.ConfigFile
					existingProj.WorkingDir = localProj.WorkingDir
				}
			}

			for _, svcName := range localProj.Services {
				svcNode := findOrCreateChild(existingProj, svcName, NodeService, stateMap)
				if svcNode.ProjectName == "" {
					svcNode.ProjectName = existingProj.Name
				}
				if svcNode.ConfigFiles == "" {
					svcNode.ConfigFiles = localProj.ConfigFile
					svcNode.WorkingDir = localProj.WorkingDir
				}
			}
		}
	}

	// Sort entire tree: projects A-Z, services A-Z, containers by running state
	slices.SortFunc(roots, func(a, b *TreeNode) int {
		return cmp.Compare(a.Name, b.Name)
	})
	for _, proj := range roots {
		slices.SortFunc(proj.Children, func(a, b *TreeNode) int {
			return cmp.Compare(a.Name, b.Name)
		})
		for _, svc := range proj.Children {
			slices.SortStableFunc(svc.Children, compareContainerNodes)
		}
	}

	if standaloneNode != nil {
		slices.SortStableFunc(standaloneNode.Children, compareContainerNodes)
		roots = append(roots, standaloneNode)
	}

	return roots
}

func Flatten(nodes []*TreeNode, parentPrefix string) []FlatItem {
	var list []FlatItem
	for i, node := range nodes {
		if node == nil {
			continue
		}

		isLast := (i == len(nodes)-1)

		var currentPrefix string
		var childPrefix string

		if node.Type == NodeProject {
			currentPrefix = ""
			childPrefix = ""
		} else {
			branch := "├─ "
			if isLast {
				branch = "└─ "
			}
			currentPrefix = parentPrefix + branch

			if isLast {
				childPrefix = parentPrefix + "   "
			} else {
				childPrefix = parentPrefix + "│  "
			}
		}

		list = append(list, FlatItem{
			Node:   node,
			Prefix: currentPrefix,
		})

		if node.Expanded && len(node.Children) > 0 {
			if node.Type == NodeProject {
				list = append(list, Flatten(node.Children, "  ")...)
			} else {
				list = append(list, Flatten(node.Children, childPrefix)...)
			}
		}
	}
	return list
}
