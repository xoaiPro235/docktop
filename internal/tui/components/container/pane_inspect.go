package container

import (
	"bytes"
	"cmp"
	"fmt"
	"slices"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/neilotoole/jsoncolor"
	"github.com/xoaiPro235/docktop/internal/tui/keys"
	"github.com/xoaiPro235/docktop/internal/tui/theme"
)

type InspectLoadedMsg struct {
	ContainerID string
	Data        *container.InspectResponse
	Err         error
}

type InspectPane struct {
	viewport  viewport.Model
	ready     bool
	width     int
	height    int
	rawJSON   bool
	currentID string
	lastData  *container.InspectResponse
}

func NewInspectPane() InspectPane {
	return InspectPane{
		rawJSON: false,
	}
}

func (p *InspectPane) SetSize(w, h int) {
	p.width = w
	p.height = h
	if !p.ready {
		p.viewport = viewport.New(viewport.WithWidth(w), viewport.WithHeight(h))
		p.ready = true
	} else {
		p.viewport.SetWidth(w)
		p.viewport.SetHeight(h)
	}
	if p.lastData != nil {
		p.SetContent(p.lastData)
	}
}

func (p *InspectPane) ToggleRaw() {
	p.rawJSON = !p.rawJSON
	if p.lastData != nil {
		p.SetContent(p.lastData)
	}
}

func (p *InspectPane) ToggleWrap() {
	p.viewport.SoftWrap = !p.viewport.SoftWrap
}

func (p *InspectPane) SetContent(inspect *container.InspectResponse) {
	if inspect == nil {
		p.viewport.SetContent("No inspect data available.")
		return
	}

	p.lastData = inspect
	p.currentID = inspect.ID

	if p.rawJSON {
		var buf bytes.Buffer
		enc := jsoncolor.NewEncoder(&buf)
		enc.SetIndent("", "  ")
		enc.SetColors(jsoncolor.DefaultColors())
		if err := enc.Encode(inspect); err != nil {
			p.viewport.SetContent("JSON parse error: " + err.Error())
			return
		}
		p.viewport.SetContent(buf.String())
		return
	}

	p.viewport.SetContent(renderFormattedInspect(inspect, p.width))
}

func (p InspectPane) Update(msg tea.Msg) (InspectPane, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keys.ContainerKeys.ToggleRaw):
			p.ToggleRaw()
			return p, nil
		case key.Matches(msg, keys.ContainerKeys.ToggleWrap):
			p.ToggleWrap()
			return p, nil
		}
	}
	p.viewport, cmd = p.viewport.Update(msg)
	return p, cmd
}

func (p InspectPane) Render() string {
	if !p.ready {
		return "Initializing inspect viewer..."
	}

	modeStr := "Formatted"
	if p.rawJSON {
		modeStr = "Raw JSON"
	}

	modePart := theme.Current.Key.Render(" Mode: ") + theme.Current.AccentText.Render(modeStr)
	keyHint := theme.Current.FooterKey.Render("[t]") + " " + theme.Current.Title.Render("Toggle Raw JSON")
	scrollPart := theme.Current.MutedText.Render(fmt.Sprintf("Scroll: %3.f%%", p.viewport.ScrollPercent()*100))
	sep := theme.Current.LineNumber.Render(" │ ")

	footer := theme.Current.Footer.
		Width(p.width).
		MaxHeight(1).
		Render(modePart + sep + keyHint + sep + scrollPart)

	return p.viewport.View() + "\n" + footer
}

func renderFormattedInspect(data *container.InspectResponse, width int) string {
	var sb strings.Builder

	sectionTitle := theme.Current.SectionTitle
	keyStyle := theme.Current.Key
	valStyle := theme.Current.Value
	accentStyle := theme.Current.AccentText

	makeHeader := func(title string) string {
		header := fmt.Sprintf("── %s ──", title)
		if fill := width - ansi.StringWidth(header) - 2; fill > 0 {
			header += strings.Repeat("─", fill)
		}
		return sectionTitle.Render(header) + "\n"
	}

	// 1. General & State
	sb.WriteString(makeHeader("GENERAL & STATE"))
	fmt.Fprintf(&sb, "  %s %s\n", keyStyle.Render("ID:          "), valStyle.Render(data.ID))
	fmt.Fprintf(&sb, "  %s %s\n", keyStyle.Render("Name:        "), accentStyle.Render(data.Name))
	fmt.Fprintf(&sb, "  %s %s\n", keyStyle.Render("Created:     "), valStyle.Render(data.Created))
	if data.State != nil {
		fmt.Fprintf(&sb, "  %s %s (ExitCode: %d, PID: %d)\n\n",
			keyStyle.Render("Status:      "),
			valStyle.Render(string(data.State.Status)),
			data.State.ExitCode,
			data.State.Pid)
	} else {
		fmt.Fprintf(&sb, "  %s %s\n\n", keyStyle.Render("Status:      "), valStyle.Render("Unknown"))
	}

	// 2. Port Mappings & Networks
	sb.WriteString(makeHeader("NETWORK & PORTS"))
	if data.NetworkSettings == nil || len(data.NetworkSettings.Ports) == 0 {
		sb.WriteString("  No port bindings\n")
	} else {
		portKeys := make([]network.Port, 0, len(data.NetworkSettings.Ports))
		for p := range data.NetworkSettings.Ports {
			portKeys = append(portKeys, p)
		}
		slices.SortFunc(portKeys, func(a, b network.Port) int {
			if a.Num() != b.Num() {
				return cmp.Compare(a.Num(), b.Num())
			}
			return cmp.Compare(a.String(), b.String())
		})

		for _, port := range portKeys {
			bindings := data.NetworkSettings.Ports[port]
			// port.String() returns standard "80/tcp" format
			portStr := port.String()

			if len(bindings) == 0 {
				fmt.Fprintf(&sb, "  %s -> (Not mapped)\n", accentStyle.Render(portStr))
			} else {
				// Sort bindings: IPv4 first, then IPv6
				slices.SortStableFunc(bindings, func(a, b network.PortBinding) int {
					if !a.HostIP.Is6() && b.HostIP.Is6() {
						return -1
					}
					if a.HostIP.Is6() && !b.HostIP.Is6() {
						return 1
					}
					return 0
				})

				for _, b := range bindings {
					var hostMap string
					if b.HostIP.Is6() {
						hostMap = fmt.Sprintf("%s:%s", b.HostIP.String(), b.HostPort)
					} else if b.HostIP.IsValid() && !b.HostIP.IsUnspecified() {
						hostMap = fmt.Sprintf("%s:%s", b.HostIP.String(), b.HostPort)
					} else {
						hostMap = b.HostPort
					}

					fmt.Fprintf(&sb, "  %s -> %s\n", accentStyle.Render(hostMap), valStyle.Render(portStr))
				}
			}
		}
	}

	sb.WriteString("\n  Networks:\n")
	if data.NetworkSettings == nil || len(data.NetworkSettings.Networks) == 0 {
		sb.WriteString("    (None)\n")
	} else {
		netNames := make([]string, 0, len(data.NetworkSettings.Networks))
		for netName := range data.NetworkSettings.Networks {
			netNames = append(netNames, netName)
		}
		slices.Sort(netNames)

		for _, netName := range netNames {
			netConfig := data.NetworkSettings.Networks[netName]
			if netConfig == nil {
				continue
			}

			ip := "-"
			if netConfig.IPAddress.IsValid() && !netConfig.IPAddress.IsUnspecified() {
				ip = netConfig.IPAddress.String()
				if netConfig.IPPrefixLen > 0 {
					ip = fmt.Sprintf("%s/%d", ip, netConfig.IPPrefixLen)
				}
			}

			gw := "-"
			if netConfig.Gateway.IsValid() && !netConfig.Gateway.IsUnspecified() {
				gw = netConfig.Gateway.String()
			}

			mac := "-"
			if len(netConfig.MacAddress) > 0 {
				mac = netConfig.MacAddress.String()
			}

			fmt.Fprintf(&sb, "    - %s\n", accentStyle.Render(netName))
			fmt.Fprintf(&sb, "        IP: %s | Gateway: %s | MAC: %s\n", valStyle.Render(ip), valStyle.Render(gw), valStyle.Render(mac))

			// IPv6 if available
			if netConfig.GlobalIPv6Address.IsValid() && !netConfig.GlobalIPv6Address.IsUnspecified() {
				fmt.Fprintf(&sb, "        IPv6: %s\n", valStyle.Render(netConfig.GlobalIPv6Address.String()))
			}

			// Internal DNS aliases in Compose network
			if len(netConfig.Aliases) > 0 {
				fmt.Fprintf(&sb, "        Aliases: %s\n", valStyle.Render(strings.Join(netConfig.Aliases, ", ")))
			}
		}
	}
	sb.WriteString("\n")

	// 3. Mounts & Volumes
	sb.WriteString(makeHeader("MOUNTS & VOLUMES"))
	if len(data.Mounts) == 0 {
		sb.WriteString("  No volumes/bind mounts\n")
	} else {
		for _, m := range data.Mounts {
			mode := "rw"
			if !m.RW {
				mode = "ro"
			}
			fmt.Fprintf(&sb, "  [%s] %s -> %s (%s)\n",
				accentStyle.Render(string(m.Type)),
				valStyle.Render(m.Source),
				accentStyle.Render(m.Destination),
				mode)
		}
	}
	sb.WriteString("\n")

	// 4. Environment Variables
	sb.WriteString(makeHeader("ENVIRONMENT VARIABLES"))
	if data.Config != nil && len(data.Config.Env) > 0 {
		envs := make([]string, len(data.Config.Env))
		copy(envs, data.Config.Env)
		slices.Sort(envs)

		for _, env := range envs {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				key, val := parts[0], parts[1]
				lowerKey := strings.ToLower(key)
				if strings.Contains(lowerKey, "secret") || strings.Contains(lowerKey, "password") || strings.Contains(lowerKey, "token") {
					val = "******** (Masked)"
				}
				fmt.Fprintf(&sb, "  %s=%s\n", keyStyle.Render(key), valStyle.Render(val))
			} else {
				fmt.Fprintf(&sb, "  %s\n", valStyle.Render(env))
			}
		}
	} else {
		sb.WriteString("  No environment variables set\n")
	}

	return sb.String()
}
