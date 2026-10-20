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
	Container   *docker.ContainerInfo
	ConfigFiles string
	WorkingDir  string
	Expanded    bool
	Children    []*TreeNode
}

func (n *TreeNode) DisplayTitle() string {
	arrow := "▼"
	if !n.Expanded {
		arrow = "▶"
	}

	switch n.Type {
	case NodeProject:
		if n.Name == StandaloneProjectName {
			return fmt.Sprintf("%s Standalone (%d)", arrow, len(n.Children))
		}
		return fmt.Sprintf("%s %s", arrow, n.Name)
	case NodeService:
		return fmt.Sprintf("%s %s (%d)", arrow, n.Name, len(n.Children))
	case NodeContainer:
		statusDot := "●"
		if n.Container != nil && n.Container.State != docker.StateRunning {
			statusDot = "○"
		}
		title := fmt.Sprintf("%s %s", statusDot, n.Name)
		if n.Container != nil && n.Container.OneOff {
			title += " [one-off]"
		}
		return title
	default:
		return n.Name
	}
}

type FlatItem struct {
	Node   *TreeNode
	Prefix string
}

func containerStateScore(state string) int {
	switch strings.ToLower(state) {
	case string(docker.StateRunning):
		return 1
	case string(docker.StateRestarting):
		return 2
	case string(docker.StatePaused):
		return 3
	case string(docker.StateCreated):
		return 4
	case string(docker.StateExited), string(docker.StateDead):
		return 5
	default:
		return 6
	}
}

func compareContainerNodes(a, b *TreeNode) int {
	scoreA := 99
	scoreB := 99
	if a.Container != nil {
		scoreA = containerStateScore(string(a.Container.State))
	}

	if b.Container != nil {
		scoreB = containerStateScore(string(b.Container.State))
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

func BuildTree(containers []docker.ContainerInfo, stateMap map[string]bool) []*TreeNode {
	var roots []*TreeNode
	var standaloneNode *TreeNode

	for i := range containers {
		c := &containers[i]

		if c.IsCompose && c.ProjectName != "" {
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
					ConfigFiles: c.ConfigFiles,
					WorkingDir:  c.WorkingDir,
					Expanded:    isNodeExpanded(key, stateMap),
				}
				roots = append(roots, projNode)
			} else {
				if projNode.ConfigFiles == "" && c.ConfigFiles != "" {
					projNode.ConfigFiles = c.ConfigFiles
					projNode.WorkingDir = c.WorkingDir
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
			if svcNode.ConfigFiles == "" && c.ConfigFiles != "" {
				svcNode.ConfigFiles = c.ConfigFiles
				svcNode.WorkingDir = c.WorkingDir
			}

			svcNode.Children = append(svcNode.Children, &TreeNode{
				Type:        NodeContainer,
				Name:        c.Name,
				ProjectName: c.ProjectName,
				Key:         "ctr:" + c.ID,
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
			standaloneNode.Children = append(standaloneNode.Children, &TreeNode{
				Type:      NodeContainer,
				Name:      c.Name,
				Key:       "ctr:" + c.ID,
				Container: c,
			})
		}
	}

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

// resolveComposeProject extracts ProjectName, ServiceName, WorkingDir, and ConfigFile for a TreeNode.
func resolveComposeProject(node *TreeNode) (projectName string, serviceName string, workDir string, configFile string) {
	if node == nil {
		return "", "", "", ""
	}
	projectName = node.ProjectName
	if projectName == "" && node.Type == NodeProject && node.Name != StandaloneProjectName {
		projectName = node.Name
	}
	if node.Type == NodeService {
		serviceName = node.Name
	}
	configFile = findComposeFile(node)
	if configFile != "" {
		workDir = filepath.Dir(configFile)
	}
	if workDir == "" {
		workDir = node.WorkingDir
	}
	if workDir == "" {
		for _, child := range node.Children {
			if child.WorkingDir != "" {
				workDir = child.WorkingDir
				break
			}
			if child.Container != nil && child.Container.WorkingDir != "" {
				workDir = child.Container.WorkingDir
				break
			}
			for _, gc := range child.Children {
				if gc.WorkingDir != "" {
					workDir = gc.WorkingDir
					break
				}
				if gc.Container != nil && gc.Container.WorkingDir != "" {
					workDir = gc.Container.WorkingDir
					break
				}
			}
		}
	}
	return projectName, serviceName, workDir, configFile
}
