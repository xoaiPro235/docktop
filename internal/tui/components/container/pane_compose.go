package container

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/xoaiPro235/docktop/internal/tui/keys"
	"github.com/xoaiPro235/docktop/internal/tui/theme"
)

type ComposePane struct {
	viewport    viewport.Model
	projectName string
	filePath    string
	content     string
	lastNode    *TreeNode
	width       int
	height      int
	ready       bool
}

func NewComposePane() ComposePane {
	return ComposePane{}
}

func (p *ComposePane) SetSize(w, h int) {
	p.width = w
	p.height = h
	if !p.ready {
		p.viewport = viewport.New(viewport.WithWidth(w), viewport.WithHeight(h))
		p.ready = true
	} else {
		p.viewport.SetWidth(w)
		p.viewport.SetHeight(h)
	}
	if p.lastNode != nil {
		p.LoadProject(p.lastNode)
	}
}

func (p *ComposePane) LoadProject(node *TreeNode) {
	if node == nil {
		p.lastNode = nil
		p.filePath = ""
		p.content = ""
		p.viewport.SetContent("No project selected.")
		return
	}

	// Case 1: Standalone containers root
	if node.Name == StandaloneProjectName {
		p.lastNode = node
		p.projectName = node.Name
		p.filePath = ""
		p.content = ""
		p.viewport.SetContent(renderStandaloneSummary(node, p.width))
		return
	}

	// Case 2: Compose project
	configPath := findComposeFile(node)

	// Avoid re-reading disk and resetting scroll position if already displaying this file
	if configPath != "" && p.filePath == configPath && p.content != "" {
		p.lastNode = node
		p.projectName = node.Name
		return
	}

	p.lastNode = node
	p.projectName = node.Name
	p.filePath = configPath

	if configPath == "" {
		p.content = ""
		p.viewport.SetContent(renderProjectSummary(node, "Compose file path not found in container labels.", p.width))
		return
	}

	bytes, err := os.ReadFile(configPath)
	if err != nil {
		p.content = ""
		p.viewport.SetContent(renderProjectSummary(node, fmt.Sprintf("Failed to read compose file (%s): %v", configPath, err), p.width))
		return
	}

	p.content = string(bytes)
	p.viewport.SetContent(highlightYAML(p.content))
}

func (p *ComposePane) ToggleWrap() {
	p.viewport.SoftWrap = !p.viewport.SoftWrap
}

func (p ComposePane) Update(msg tea.Msg) (ComposePane, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keys.ContainerKeys.ToggleWrap):
			p.ToggleWrap()
			return p, nil
		}
	}
	p.viewport, cmd = p.viewport.Update(msg)
	return p, cmd
}

func (p ComposePane) Render() string {
	if !p.ready {
		return "Initializing compose viewer..."
	}

	pathInfo := theme.Current.Key.Render(" Path: ") + theme.Current.Value.Render(p.filePath)
	sep := theme.Current.LineNumber.Render(" │ ")
	scrollInfo := theme.Current.MutedText.Render(fmt.Sprintf("Scroll: %3.f%%", p.viewport.ScrollPercent()*100))

	footer := theme.Current.Footer.
		Width(p.width).
		MaxHeight(1).
		Render(pathInfo + sep + scrollInfo)

	return p.viewport.View() + "\n" + footer
}

func resolveExistingFile(path string) string {
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

func checkConfigFiles(configFiles string) string {
	for f := range strings.SplitSeq(configFiles, ",") {
		if resolved := resolveExistingFile(f); resolved != "" {
			return resolved
		}
	}
	return ""
}

func checkWorkingDir(dir string) string {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return ""
	}
	candidates := []string{
		filepath.Join(dir, "docker-compose.yml"),
		filepath.Join(dir, "docker-compose.yaml"),
		filepath.Join(dir, "compose.yml"),
		filepath.Join(dir, "compose.yaml"),
	}
	for _, c := range candidates {
		if resolved := resolveExistingFile(c); resolved != "" {
			return resolved
		}
	}
	return ""
}

func findComposeFile(node *TreeNode) string {
	if node == nil {
		return ""
	}
	if node.ConfigFiles != "" {
		if f := checkConfigFiles(node.ConfigFiles); f != "" {
			return f
		}
	}

	if node.WorkingDir != "" {
		if f := checkWorkingDir(node.WorkingDir); f != "" {
			return f
		}
	}

	// Try children containers' labels if root node didn't have it
	for _, child := range node.Children {
		if child.Container != nil {
			if child.Container.ConfigFiles != "" {
				if f := checkConfigFiles(child.Container.ConfigFiles); f != "" {
					return f
				}
			}
			if child.Container.WorkingDir != "" {
				if f := checkWorkingDir(child.Container.WorkingDir); f != "" {
					return f
				}
			}
		}
		for _, grandchild := range child.Children {
			if grandchild.Container != nil {
				if grandchild.Container.ConfigFiles != "" {
					if f := checkConfigFiles(grandchild.Container.ConfigFiles); f != "" {
						return f
					}
				}
				if grandchild.Container.WorkingDir != "" {
					if f := checkWorkingDir(grandchild.Container.WorkingDir); f != "" {
						return f
					}
				}
			}
		}
	}

	// Fallback: check current directory
	candidates := []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"}
	for _, c := range candidates {
		if resolved := resolveExistingFile(c); resolved != "" {
			if abs, err := filepath.Abs(resolved); err == nil {
				return abs
			}
			return resolved
		}
	}

	return ""
}

func highlightYAML(content string) string {
	lines := strings.Split(content, "\n")
	var sb strings.Builder

	keyStyle := lipgloss.NewStyle().Foreground(theme.Current.Palette.Secondary)
	valStyle := theme.Current.Title
	commentStyle := theme.Current.Comment
	lineNumStyle := theme.Current.LineNumber
	dashStyle := theme.Current.FooterKey

	maxDigits := len(fmt.Sprintf("%d", len(lines)))
	lineFormat := fmt.Sprintf("%%%dd │ ", maxDigits)

	for i, line := range lines {
		numPrefix := lineNumStyle.Render(fmt.Sprintf(lineFormat, i+1))
		sb.WriteString(numPrefix)

		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "#") {
			sb.WriteString(commentStyle.Render(line))
		} else if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			keyPart := parts[0]
			valPart := parts[1]

			if strings.Contains(keyPart, "- ") {
				subParts := strings.SplitN(keyPart, "- ", 2)
				sb.WriteString(subParts[0])
				sb.WriteString(dashStyle.Render("- "))
				sb.WriteString(keyStyle.Render(subParts[1] + ":"))
			} else {
				sb.WriteString(keyStyle.Render(keyPart + ":"))
			}

			if strings.TrimSpace(valPart) != "" {
				trimmedVal := strings.TrimSpace(valPart)
				if strings.HasPrefix(trimmedVal, "#") {
					sb.WriteString(" ")
					sb.WriteString(commentStyle.Render(trimmedVal))
				} else {
					sb.WriteString(" ")
					sb.WriteString(valStyle.Render(trimmedVal))
				}
			}
		} else if strings.HasPrefix(trimmed, "- ") {
			indentLen := len(line) - len(strings.TrimLeft(line, " "))
			indent := strings.Repeat(" ", indentLen)
			rest := strings.TrimPrefix(trimmed, "- ")
			sb.WriteString(indent)
			sb.WriteString(dashStyle.Render("- "))
			sb.WriteString(valStyle.Render(rest))
		} else {
			sb.WriteString(valStyle.Render(line))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func renderProjectSummary(node *TreeNode, extraNote string, width int) string {
	if node == nil {
		return ""
	}
	var sb strings.Builder
	titleStyle := theme.Current.TreeProject
	labelStyle := theme.Current.Key
	valStyle := theme.Current.Title

	header := fmt.Sprintf("── COMPOSE PROJECT: %s ──", node.Name)
	if fill := width - ansi.StringWidth(header) - 2; fill > 0 {
		header += strings.Repeat("─", fill)
	}
	sb.WriteString(titleStyle.Render(header))
	sb.WriteString("\n\n")

	if extraNote != "" {
		sb.WriteString(lipgloss.NewStyle().Foreground(theme.Current.Palette.Warning).Render(extraNote + "\n\n"))
	}

	sb.WriteString(labelStyle.Render("Services:\n"))
	for _, svc := range node.Children {
		fmt.Fprintf(&sb, "  • %s (%d containers)\n", valStyle.Render(svc.Name), len(svc.Children))
		for _, ctrNode := range svc.Children {
			if ctrNode.Container != nil {
				status := RenderStatusBadge(string(ctrNode.Container.State))
				fmt.Fprintf(&sb, "    - %s  [%s]  Image: %s\n",
					valStyle.Render(ctrNode.Name),
					status,
					ctrNode.Container.Image)
			}
		}
	}

	return sb.String()
}

func renderStandaloneSummary(node *TreeNode, width int) string {
	if node == nil {
		return ""
	}
	var sb strings.Builder
	titleStyle := theme.Current.TreeProject
	valStyle := theme.Current.Title

	header := "── STANDALONE CONTAINERS ──"
	if fill := width - ansi.StringWidth(header) - 2; fill > 0 {
		header += strings.Repeat("─", fill)
	}
	sb.WriteString(titleStyle.Render(header))
	sb.WriteString("\n\n")
	fmt.Fprintf(&sb, "Total Containers: %d\n\n", len(node.Children))

	for _, ctrNode := range node.Children {
		if ctrNode.Container != nil {
			c := ctrNode.Container
			status := RenderStatusBadge(string(c.State))
			fmt.Fprintf(&sb, "• %s  (%s)  [%s]\n", valStyle.Render(c.Name), c.ShortID(), status)
			fmt.Fprintf(&sb, "    Image: %s\n", c.Image)
			if c.Ports != "" {
				fmt.Fprintf(&sb, "    Ports: %s\n", c.Ports)
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}
