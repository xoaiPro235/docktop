package image

import (
	"fmt"
	"image/color"
	"runtime"
	"strings"

	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/xoaiPro235/docktop/internal/docker"
	"github.com/xoaiPro235/docktop/internal/tui/theme"
)

// HighlightCommand highlights Dockerfile instructions with dedicated syntax colors.
func HighlightCommand(cmd string, maxWidth int) string {
	words := strings.Fields(cmd)
	if len(words) == 0 {
		return ""
	}

	keyword := words[0]
	upper := strings.ToUpper(keyword)

	var kwColor color.Color
	switch upper {
	case "RUN":
		kwColor = theme.Current.Palette.Primary
	case "COPY", "ADD":
		kwColor = theme.Current.Palette.Success
	case "CMD", "ENTRYPOINT":
		kwColor = theme.Current.Palette.Warning
	case "ENV", "ARG":
		kwColor = theme.Current.Palette.Accent
	case "EXPOSE":
		kwColor = theme.Current.Palette.Secondary
	case "WORKDIR", "USER", "VOLUME", "LABEL", "HEALTHCHECK":
		kwColor = theme.Current.Palette.Highlight
	default:
		kwColor = theme.Current.Palette.Foreground
	}

	kwStyled := lipgloss.NewStyle().Bold(true).Foreground(kwColor).Render(keyword)
	rest := ""
	if len(words) > 1 {
		rest = " " + strings.Join(words[1:], " ")
	}

	full := kwStyled + theme.Current.Value.Render(rest)
	if maxWidth > 0 {
		return ansi.Truncate(full, maxWidth, "…")
	}
	return full
}

// FormatImageDetailsContent formats the entire right panel (metadata + layer history) for the scrollable viewport.
func FormatImageDetailsContent(selected *docker.ImageInfo, width int) string {
	if selected == nil {
		return theme.Current.MutedText.Render("No image selected")
	}

	var sb strings.Builder

	// 1. Details Section Header
	sb.WriteString(theme.Current.SectionHeader("IMAGE DETAILS", width))

	// Full ID (not truncated, allows horizontal scroll)
	fmt.Fprintf(&sb, "%s %s\n", theme.Current.Key.Render("ID:         "), theme.Current.Value.Render(selected.ID))

	// Created
	createdStr := docker.FormatTimeAgo(selected.Created)
	if !selected.Created.IsZero() {
		createdStr = fmt.Sprintf("%s (%s)", docker.FormatTimeAgo(selected.Created), selected.Created.Format("2006-01-02 15:04"))
	}
	fmt.Fprintf(&sb, "%s %s\n", theme.Current.Key.Render("Created:    "), theme.Current.Value.Render(createdStr))

	// OS / Architecture
	osArch := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
	if selected.Details != nil && selected.Details.Os != "" {
		osArch = fmt.Sprintf("%s/%s", selected.Details.Os, selected.Details.Architecture)
		if selected.Details.Variant != "" {
			osArch += "/" + selected.Details.Variant
		}
	}
	fmt.Fprintf(&sb, "%s %s\n", theme.Current.Key.Render("OS/Arch:    "), theme.Current.Value.Render(osArch))

	// Size
	sizeStr := fmt.Sprintf("%s (%d bytes)", docker.FormatImageSize(selected.Size), selected.Size)
	fmt.Fprintf(&sb, "%s %s\n", theme.Current.Key.Render("Size:       "), theme.Current.Value.Render(sizeStr))

	// Containers using this
	sb.WriteString(theme.Current.Key.Render("Containers using this:"))
	sb.WriteString("\n")
	if len(selected.UsedByContainers) == 0 {
		sb.WriteString(theme.Current.MutedText.Render("  (none - unused)"))
		sb.WriteString("\n")
	} else {
		for _, cName := range selected.UsedByContainers {
			fmt.Fprintf(&sb, "  %s %s\n", theme.Current.TreeBranch.Render("-"), theme.Current.AccentText.Render(cName))
		}
	}

	sb.WriteString("\n")

	// 2. Layers (History) Section Header
	layersHeader := "IMAGE LAYERS (HISTORY)"
	if len(selected.Layers) > 0 {
		layersHeader = fmt.Sprintf("IMAGE LAYERS (HISTORY) [%d layers]", len(selected.Layers))
	}
	sb.WriteString(theme.Current.SectionHeader(layersHeader, width))

	// Table columns: # (3 or 4), SIZE (10), COMMAND
	colNumWidth := 3
	if len(selected.Layers) >= 100 {
		colNumWidth = 4
	}
	colSizeWidth := 10

	// Table Header (always aligned with 2-char cursor space so # aligns with line numbers)
	headerNum := theme.Current.TreeHeader.Render(fmt.Sprintf("%-*s", colNumWidth, "#"))
	headerSize := theme.Current.TreeHeader.Render(fmt.Sprintf("%-*s", colSizeWidth, "SIZE"))
	headerCmd := theme.Current.TreeHeader.Render("COMMAND")
	fmt.Fprintf(&sb, "  %s %s %s\n", headerNum, headerSize, headerCmd)

	// 3. Layer rows (full command un-truncated to allow horizontal scroll)
	for _, layer := range selected.Layers {
		numStr := fmt.Sprintf("%-*d", colNumWidth, layer.Index)
		sizeStr := fmt.Sprintf("%-*s", colSizeWidth, docker.FormatImageSize(selected.Size))
		cmdStr := HighlightCommand(layer.Command, 0)
		fmt.Fprintf(&sb, "  %s %s %s\n",
			theme.Current.LineNumber.Render(numStr),
			theme.Current.Value.Render(sizeStr),
			cmdStr)
	}

	return strings.TrimRight(sb.String(), "\n")
}

// RenderDetailsAndLayers renders the complete right panel containing metadata and layer history viewport.
func RenderDetailsAndLayers(selected *docker.ImageInfo, vp *viewport.Model, width, height int, isFocused bool) string {
	if vp != nil {
		return vp.View()
	}
	return FormatImageDetailsContent(selected, width)
}
