package volume

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"

	"charm.land/bubbles/v2/viewport"
	"github.com/charmbracelet/x/ansi"
	"github.com/docker/go-units"
	"github.com/xoaiPro235/docktop/internal/docker"
	"github.com/xoaiPro235/docktop/internal/tui/theme"
)

// RenderVolumeDetails renders the right panel using the scrollable viewport.
func RenderVolumeDetails(selected *docker.VolumeInfo, vp *viewport.Model, width, height int, isFocused bool) string {
	if vp != nil {
		return vp.View()
	}
	return FormatVolumeViewportContent(selected, width)
}

// FormatVolumeViewportContent formats the complete volume details, labels, and mounted containers for the scrollable viewport.
func FormatVolumeViewportContent(selected *docker.VolumeInfo, width int) string {
	if selected == nil {
		return theme.Current.MutedText.Render("No volume selected.")
	}

	var sb strings.Builder

	makeHeader := func(title string) string {
		header := fmt.Sprintf("── %s ──", title)
		if fill := width - ansi.StringWidth(header) - 2; fill > 0 {
			header += strings.Repeat("─", fill)
		}
		return theme.Current.SectionTitle.Render(header) + "\n"
	}

	// 1. Details Section
	sb.WriteString(makeHeader("VOLUME DETAILS"))

	// Volume Name
	sb.WriteString(fmt.Sprintf("%s %s\n", theme.Current.Key.Render("Name:       "), theme.Current.AccentText.Render(selected.Name)))

	// Driver
	sb.WriteString(fmt.Sprintf("%s %s\n", theme.Current.Key.Render("Driver:     "), theme.Current.Value.Render(selected.Driver)))

	// Scope
	sb.WriteString(fmt.Sprintf("%s %s\n", theme.Current.Key.Render("Scope:      "), theme.Current.Value.Render(selected.Scope)))

	// Created
	var createdAt time.Time
	if selected.CreatedAt != "" {
		// RFC3339 or ISO8601
		t, err := time.Parse(time.RFC3339Nano, selected.CreatedAt)
		if err == nil {
			createdAt = t
		} else {
			t, err = time.Parse(time.RFC3339, selected.CreatedAt)
			if err == nil {
				createdAt = t
			}
		}
	}

	var formattedTime string
	if !createdAt.IsZero() {
		formattedTime = docker.FormatTimeAgo(createdAt)
	} else {
		formattedTime = "-"
	}

	createdDisplay := formattedTime
	sb.WriteString(fmt.Sprintf("%s %s\n", theme.Current.Key.Render("Created:    "), theme.Current.Value.Render(createdDisplay)))

	// Size
	var formattedSize string = "-"
	if selected.UsageData != nil && selected.UsageData.Size >= 0 {
		size := selected.UsageData.Size
		formattedSize = units.HumanSize(float64(size))
	}
	sizeDisplay := formattedSize

	sb.WriteString(fmt.Sprintf("%s %s\n", theme.Current.Key.Render("Size:       "), theme.Current.StatusRunning.Render(sizeDisplay)))

	// Mountpoint
	mountDisplay := selected.Mountpoint
	if mountDisplay == "" {
		mountDisplay = "-"
	}
	sb.WriteString(fmt.Sprintf("%s %s\n\n", theme.Current.Key.Render("Mountpoint: "), theme.Current.MutedText.Render(mountDisplay)))

	// 2. Labels Section
	labelCount := len(selected.Labels)
	labelsHeader := fmt.Sprintf("LABELS (%d)", labelCount)
	sb.WriteString(makeHeader(labelsHeader))

	if labelCount == 0 {
		sb.WriteString("  ")
		sb.WriteString(theme.Current.MutedText.Render("(no labels)"))
		sb.WriteString("\n\n")
	} else {
		// Sort keys alphabetically to guarantee 100% stable, deterministic order
		keys := make([]string, 0, labelCount)
		for k := range selected.Labels {
			keys = append(keys, k)
		}
		slices.Sort(keys)

		for _, k := range keys {
			val := selected.Labels[k]
			lineStr := fmt.Sprintf("  %s %s", theme.Current.AccentText.Render(k+":"), theme.Current.Value.Render(val))
			sb.WriteString(lineStr)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	// 3. Mounted Containers Section
	containerCount := len(selected.Containers)
	containersHeader := fmt.Sprintf("MOUNTED BY CONTAINERS (%d)", containerCount)
	sb.WriteString(makeHeader(containersHeader))

	if containerCount == 0 {
		sb.WriteString("  ")
		sb.WriteString(theme.Current.StatusPaused.Render("○ No containers currently mounting this volume"))
		sb.WriteString("\n")
		sb.WriteString("  ")
		sb.WriteString(theme.Current.MutedText.Render("(Safe to delete or prune)"))
		sb.WriteString("\n")
	} else {
		slices.SortStableFunc(selected.Containers, func(a, b docker.VolumeContainerMount) int {
			return cmp.Compare(a.ContainerName, b.ContainerName)
		})

		for _, c := range selected.Containers {
			var stateBadge string
			switch strings.ToLower(string(c.State)) {
			case string(docker.StateRunning):
				stateBadge = theme.Current.StatusRunning.Render("running")
			case string(docker.StatePaused):
				stateBadge = theme.Current.StatusPaused.Render("paused")
			case string(docker.StateRestarting):
				stateBadge = theme.Current.StatusPaused.Render("restarting")
			default:
				displayState := c.State
				if displayState == "" {
					displayState = "stopped"
				}
				stateBadge = theme.Current.StatusStopped.Render(string(displayState))
			}

			modeText := "Read-Write (rw)"
			if !c.RW || strings.Contains(strings.ToLower(c.Mode), "ro") {
				modeText = "Read-Only (ro)"
			}

			sb.WriteString(fmt.Sprintf("  - %s (%s)\n", theme.Current.AccentText.Render(c.ContainerName), stateBadge))
			sb.WriteString(fmt.Sprintf("      Destination: %s\n", theme.Current.Value.Render(c.Destination)))
			sb.WriteString(fmt.Sprintf("      Access Mode: %s\n\n", theme.Current.Key.Render(modeText)))
		}
	}

	return strings.TrimRight(sb.String(), "\n")
}

// FormatMountedContainers is kept for backwards compatibility.
func FormatMountedContainers(selected *docker.VolumeInfo, width int) string {
	return FormatVolumeViewportContent(selected, width)
}
