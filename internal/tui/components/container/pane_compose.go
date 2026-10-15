package container

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
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
	vpHeight := max(h-2, 1)
	if !p.ready {
		p.viewport = viewport.New(viewport.WithWidth(w), viewport.WithHeight(vpHeight))
		p.ready = true
	} else {
		p.viewport.SetWidth(w)
		p.viewport.SetHeight(vpHeight)
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
	if node.Name == "__standalone__" {
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

func (p ComposePane) Update(msg tea.Msg) (ComposePane, tea.Cmd) {
	var cmd tea.Cmd
	p.viewport, cmd = p.viewport.Update(msg)
	return p, cmd
}

func (p ComposePane) Render() string {
	if !p.ready {
		return "Initializing compose viewer..."
	}

	title := p.projectName
	if p.filePath != "" {
		title = fmt.Sprintf("%s (%s)", p.projectName, filepath.Base(p.filePath))
	}

	rawHeader := fmt.Sprintf("── DOCKER COMPOSE CONFIG: %s ──", title)
	if fill := p.width - ansi.StringWidth(rawHeader) - 2; fill > 0 {
		rawHeader += strings.Repeat("─", fill)
	}
	header := theme.Current.TreeProject.Render(rawHeader)

	pathInfo := theme.Current.Key.Render(" Path: ") + theme.Current.Value.Render(p.filePath)
	sep := theme.Current.LineNumber.Render(" │ ")
	scrollInfo := theme.Current.MutedText.Render(fmt.Sprintf("Scroll: %3.f%%", p.viewport.ScrollPercent()*100))

	footer := theme.Current.Footer.
		Width(p.width).
		MaxHeight(1).
		Render(pathInfo + sep + scrollInfo)

	return header + "\n" + p.viewport.View() + "\n" + footer
}

func findComposeFile(node *TreeNode) string {
	if node.ConfigFiles != "" {
		// Can be multiple comma-separated files
		files := strings.Split(node.ConfigFiles, ",")
		if len(files) > 0 {
			first := strings.TrimSpace(files[0])
			if _, err := os.Stat(first); err == nil {
				return first
			}
		}
	}

	// Try working dir
	if node.WorkingDir != "" {
		candidates := []string{
			filepath.Join(node.WorkingDir, "docker-compose.yml"),
			filepath.Join(node.WorkingDir, "docker-compose.yaml"),
			filepath.Join(node.WorkingDir, "compose.yml"),
			filepath.Join(node.WorkingDir, "compose.yaml"),
		}
		for _, c := range candidates {
			if _, err := os.Stat(c); err == nil {
				return c
			}
		}
	}

	// Try children containers' labels if root node didn't have it
	for _, child := range node.Children {
		if child.Container != nil {
			if child.Container.ConfigFiles != "" {
				files := strings.Split(child.Container.ConfigFiles, ",")
				if len(files) > 0 {
					first := strings.TrimSpace(files[0])
					if _, err := os.Stat(first); err == nil {
						return first
					}
				}
			}
			if child.Container.WorkingDir != "" {
				candidates := []string{
					filepath.Join(child.Container.WorkingDir, "docker-compose.yml"),
					filepath.Join(child.Container.WorkingDir, "docker-compose.yaml"),
					filepath.Join(child.Container.WorkingDir, "compose.yml"),
					filepath.Join(child.Container.WorkingDir, "compose.yaml"),
				}
				for _, c := range candidates {
					if _, err := os.Stat(c); err == nil {
						return c
					}
				}
			}
		}
		for _, grandchild := range child.Children {
			if grandchild.Container != nil && grandchild.Container.ConfigFiles != "" {
				files := strings.Split(grandchild.Container.ConfigFiles, ",")
				if len(files) > 0 {
					first := strings.TrimSpace(files[0])
					if _, err := os.Stat(first); err == nil {
						return first
					}
				}
			}
		}
	}

	// Fallback: check current directory
	candidates := []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			if abs, err := filepath.Abs(c); err == nil {
				return abs
			}
			return c
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
