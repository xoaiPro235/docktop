package network

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	"github.com/charmbracelet/x/ansi"
	"github.com/xoaiPro235/docktop/internal/docker"
	"github.com/xoaiPro235/docktop/internal/tui/theme"
)

// RenderNetworkDetails renders the right panel using the scrollable viewport.
func RenderNetworkDetails(selected *docker.NetworkInfo, vp *viewport.Model, width, height int, isFocused bool) string {
	if vp != nil {
		return vp.View()
	}
	return FormatNetworkViewportContent(selected, width)
}

// FormatNetworkViewportContent formats the full network topology and connected endpoints into a scrollable string.
func FormatNetworkViewportContent(selected *docker.NetworkInfo, width int) string {
	if selected == nil {
		return theme.Current.MutedText.Render("No network selected")
	}

	var sb strings.Builder

	makeHeader := func(title string) string {
		header := fmt.Sprintf("── %s ──", title)
		if fill := width - ansi.StringWidth(header) - 2; fill > 0 {
			header += strings.Repeat("─", fill)
		}
		return theme.Current.SectionTitle.Render(header) + "\n"
	}

	// 1. Details Section Header
	sb.WriteString(makeHeader("NETWORK TOPOLOGY & SUBNET"))

	// Name
	fmt.Fprintf(&sb, "%s %s\n", theme.Current.Key.Render("Name:       "), theme.Current.AccentText.Render(selected.Name))

	// ID
	fmt.Fprintf(&sb, "%s %s\n", theme.Current.Key.Render("ID:         "), theme.Current.Value.Render(docker.ShortID(selected.ID)))

	// Driver
	fmt.Fprintf(&sb, "%s %s\n", theme.Current.Key.Render("Driver:     "), theme.Current.Value.Render(selected.Driver))

	// Scope
	fmt.Fprintf(&sb, "%s %s\n", theme.Current.Key.Render("Scope:      "), theme.Current.Value.Render(selected.Scope))

	// Subnet
	subnetStr := selected.Subnet
	if subnetStr == "" {
		subnetStr = "-"
	}
	fmt.Fprintf(&sb, "%s %s\n", theme.Current.Key.Render("Subnet:     "), theme.Current.Value.Render(subnetStr))

	// Gateway
	gwStr := selected.Gateway
	if gwStr == "" {
		gwStr = "-"
	}
	fmt.Fprintf(&sb, "%s %s\n", theme.Current.Key.Render("Gateway:    "), theme.Current.Value.Render(gwStr))

	// Internal
	internalDesc := "false (Internet Access)"
	if selected.Internal {
		internalDesc = "true (Isolated Network)"
	}
	fmt.Fprintf(&sb, "%s %s\n\n", theme.Current.Key.Render("Internal:   "), theme.Current.Value.Render(internalDesc))

	// 2. Connected Endpoints Header
	endpointsTitle := fmt.Sprintf("CONNECTED ENDPOINTS (%d)", len(selected.Containers))
	sb.WriteString(makeHeader(endpointsTitle))

	if len(selected.Containers) == 0 {
		if selected.IsSystem {
			sb.WriteString("  ")
			sb.WriteString(theme.Current.MutedText.Render("(System network - Default Docker driver)"))
			sb.WriteString("\n")
		} else {
			sb.WriteString("  ")
			sb.WriteString(theme.Current.MutedText.Render("○ No containers currently connected"))
			sb.WriteString("\n")
		}
	} else {
		ipColW := 17
		portColW := 10
		nameColW := max(width-ipColW-portColW-8, 14)

		headerIP := theme.Current.TreeHeader.Render(fmt.Sprintf("%-*s", ipColW, "IP ADDRESS"))
		headerName := theme.Current.TreeHeader.Render(fmt.Sprintf("%-*s", nameColW, "CONTAINER NAME"))
		headerPorts := theme.Current.TreeHeader.Render(fmt.Sprintf("%-*s", portColW, "PORTS"))
		fmt.Fprintf(&sb, "  %s  %s  %s\n", headerIP, headerName, headerPorts)

		tableSepWidth := width - 4
		if minW := ipColW + nameColW + portColW + 4; tableSepWidth < minW {
			tableSepWidth = minW
		}
		sb.WriteString("  ")
		sb.WriteString(theme.Current.LineNumber.Render(strings.Repeat("─", tableSepWidth)))
		sb.WriteString("\n")

		for _, ep := range selected.Containers {
			ipDisplay := theme.Current.AccentText.Render(fmt.Sprintf("%-*s", ipColW, ep.IP))
			nameDisplay := theme.Current.TreeSelected.Render(fmt.Sprintf("%-*s", nameColW, ep.Name))
			portDisplay := theme.Current.Value.Render(fmt.Sprintf("%-*s", portColW, ep.Ports))

			fmt.Fprintf(&sb, "  %s  %s  %s\n", ipDisplay, nameDisplay, portDisplay)
		}
	}

	return strings.TrimRight(sb.String(), "\n")
}

// FormatConnectedEndpoints is kept for backwards compatibility.
func FormatConnectedEndpoints(selected *docker.NetworkInfo, width int) string {
	return FormatNetworkViewportContent(selected, width)
}
