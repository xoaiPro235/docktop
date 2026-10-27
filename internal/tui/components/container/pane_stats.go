// Layout conceptsaand graph inspired by btop (https://github.com/aristocratos/btop).
package container

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/docker/go-units"
	"github.com/xoaiPro235/docktop/internal/docker"
	"github.com/xoaiPro235/docktop/internal/tui/theme"
)

// GraphDirection defines whether braille dots grow from bottom-up or top-down.
type GraphDirection int

const (
	GraphDirectionUp GraphDirection = iota
	GraphDirectionDown
)

type GraphMode int

const (
	GraphModeMirrored GraphMode = iota // CPU: Up, Mem: Down
	GraphModeBothUp                    // CPU: Up, Mem: Up
	GraphModeBothDown                  // CPU: Down, Mem: Down
	GraphModeInverted                  // CPU: Down, Mem: Up
)

func (m GraphMode) CPUDirection() GraphDirection {
	switch m {
	case GraphModeBothDown, GraphModeInverted:
		return GraphDirectionDown
	default:
		return GraphDirectionUp
	}
}

func (m GraphMode) MemDirection() GraphDirection {
	switch m {
	case GraphModeBothUp, GraphModeInverted:
		return GraphDirectionUp
	default:
		return GraphDirectionDown
	}
}

func (m GraphMode) Label() string {
	switch m {
	case GraphModeMirrored:
		return "Mirrored (CPU ▲ | Mem ▼)"
	case GraphModeBothUp:
		return "Both Up (CPU ▲ | Mem ▲)"
	case GraphModeBothDown:
		return "Both Down (CPU ▼ | Mem ▼)"
	case GraphModeInverted:
		return "Inverted (CPU ▼ | Mem ▲)"
	default:
		return "Mirrored"
	}
}

var brailleUpSymbols = [25]string{
	" ", "⢀", "⢠", "⢰", "⢸",
	"⡀", "⣀", "⣠", "⣰", "⣸",
	"⡄", "⣄", "⣤", "⣴", "⣼",
	"⡆", "⣆", "⣦", "⣶", "⣾",
	"⡇", "⣇", "⣧", "⣷", "⣿",
}

var brailleDownSymbols = [25]string{
	" ", "⠈", "⠘", "⠸", "⢸",
	"⠁", "⠉", "⠙", "⠹", "⢹",
	"⠃", "⠋", "⠛", "⠻", "⢻",
	"⠇", "⠏", "⠟", "⠿", "⢿",
	"⡇", "⡏", "⡟", "⡿", "⣿",
}

type StatsPane struct {
	cpuHistory  []float64 // CPU history %
	memHistory  []float64 // Memory history MB
	memUsage    float64   // MB
	memLimit    float64   // MB
	hasMemLimit bool      // true if user configured explicit memory quota
	memPercent  float64   // %
	netRx       float64   // Total bytes received
	netTx       float64   // Total bytes sent
	blockRead   uint64    // Total bytes read from disk
	blockWrite  uint64    // Total bytes written to disk
	graphMode   GraphMode
}

func NewStatsPane() StatsPane {
	return StatsPane{
		cpuHistory: make([]float64, 0, 120),
		memHistory: make([]float64, 0, 120),
		graphMode:  GraphModeMirrored,
	}
}

func (s *StatsPane) AddCPUData(val float64, maxPoints int) {
	if maxPoints <= 0 {
		maxPoints = 60
	}
	if len(s.cpuHistory) >= maxPoints {
		shift := len(s.cpuHistory) - maxPoints + 1
		copy(s.cpuHistory, s.cpuHistory[shift:])
		s.cpuHistory = s.cpuHistory[:maxPoints-1]
		s.cpuHistory = append(s.cpuHistory, val)
	} else {
		s.cpuHistory = append(s.cpuHistory, val)
	}
}

func (s *StatsPane) AddMemData(val float64, maxPoints int) {
	if maxPoints <= 0 {
		maxPoints = 60
	}
	if len(s.memHistory) >= maxPoints {
		shift := len(s.memHistory) - maxPoints + 1
		copy(s.memHistory, s.memHistory[shift:])
		s.memHistory = s.memHistory[:maxPoints-1]
		s.memHistory = append(s.memHistory, val)
	} else {
		s.memHistory = append(s.memHistory, val)
	}
}

func (s *StatsPane) CycleGraphMode() {
	s.graphMode = (s.graphMode + 1) % 4
}

func (s *StatsPane) ToggleGraphMode() {
	s.CycleGraphMode()
}

func (s *StatsPane) ToggleMemDirection() {
	s.CycleGraphMode()
}

func (s *StatsPane) GraphMode() GraphMode {
	return s.graphMode
}

func (s *StatsPane) CPUDirection() GraphDirection {
	return s.graphMode.CPUDirection()
}

func (s *StatsPane) MemDirection() GraphDirection {
	return s.graphMode.MemDirection()
}

func (s *StatsPane) SetMem(usage, limit float64, hasLimit bool) {
	s.memUsage = usage
	s.memLimit = limit
	s.hasMemLimit = hasLimit
	if limit > 0 {
		s.memPercent = (usage / limit) * 100.0
	}
}

func (s *StatsPane) SetIO(netRx, netTx float64, blkRead, blkWrite uint64) {
	s.netRx = netRx
	s.netTx = netTx
	s.blockRead = blkRead
	s.blockWrite = blkWrite
}

func (s *StatsPane) Reset() {
	s.cpuHistory = s.cpuHistory[:0]
	s.memHistory = s.memHistory[:0]
	s.memUsage = 0
	s.memLimit = 0
	s.hasMemLimit = false
	s.memPercent = 0
	s.netRx = 0
	s.netTx = 0
	s.blockRead = 0
	s.blockWrite = 0
}

func (s StatsPane) Render(selected *docker.ContainerInfo, width, height int, actionLoading, actionType, spinnerView string) string {
	if selected == nil {
		return theme.Current.MutedText.Padding(2).Render("No container selected.")
	}

	var b strings.Builder

	var statusBadge string
	switch actionType {
	case "restart":
		statusBadge = theme.Current.StatusPaused.Render(spinnerView + " RESTARTS")
	case "stop":
		statusBadge = theme.Current.StatusStopped.Render(spinnerView + " STOPPING")
	case "start":
		statusBadge = theme.Current.StatusRunning.Render(spinnerView + " STARTING")
	case "pause":
		statusBadge = theme.Current.StatusPaused.Render(spinnerView + " PAUSING")
	case "unpause":
		statusBadge = theme.Current.StatusRunning.Render(spinnerView + " RESUMING")
	default:
		statusBadge = RenderStatusBadge(string(selected.State))
	}

	header := fmt.Sprintf("%s  %s  %s",
		theme.Current.Title.Render(selected.Name),
		theme.Current.MutedText.Render("("+selected.ShortID()+")"),
		statusBadge,
	)
	b.WriteString(header)
	b.WriteString("\n\n")

	// CPU graph
	graphWidth := max(width-4, 10)
	availableForGraphs := height - 14
	graphHeight := max(availableForGraphs/2, 3)

	currentCPU := 0.0
	peakCPU := 0.0

	// Calculate peak only across visible points so expired spikes scroll away cleanly
	totalPointsNeeded := graphWidth * 2
	startIdx := max(len(s.cpuHistory)-totalPointsNeeded, 0)
	for _, val := range s.cpuHistory[startIdx:] {
		if val > peakCPU {
			peakCPU = val
		}
	}
	if len(s.cpuHistory) > 0 {
		currentCPU = s.cpuHistory[len(s.cpuHistory)-1]
	}

	// Auto-scale based on observed peak value
	var scaleMax float64
	switch {
	case peakCPU <= 1.0:
		scaleMax = 1.0
	case peakCPU <= 2.5:
		scaleMax = 2.5
	case peakCPU <= 5.0:
		scaleMax = 5.0
	case peakCPU <= 10.0:
		scaleMax = 10.0
	case peakCPU <= 25.0:
		scaleMax = 25.0
	case peakCPU <= 50.0:
		scaleMax = 50.0
	case peakCPU <= 100.0:
		scaleMax = 100.0
	default:
		scaleMax = math.Ceil(peakCPU/50.0) * 50.0
	}

	cpuDir := s.graphMode.CPUDirection()
	memDir := s.graphMode.MemDirection()

	cpuDirSymbol := "▲"
	if cpuDir == GraphDirectionDown {
		cpuDirSymbol = "▼"
	}

	var cpuTitle string
	if scaleMax <= 5.0 {
		cpuTitle = fmt.Sprintf("CPU: %.2f%%  [Scale: %.1f%% | Peak: %.1f%%]  [%s 'b']", currentCPU, scaleMax, peakCPU, cpuDirSymbol)
	} else {
		cpuTitle = fmt.Sprintf("CPU: %.2f%%  [Scale: %.0f%% | Peak: %.1f%%]  [%s 'b']", currentCPU, scaleMax, peakCPU, cpuDirSymbol)
	}
	b.WriteString(theme.Current.SectionTitle.Render(cpuTitle))
	b.WriteString("\n")

	rawLines := renderBtopBraille(s.cpuHistory, graphWidth, graphHeight, scaleMax, cpuDir)

	for i, line := range rawLines {
		var t float64
		if len(rawLines) > 1 {
			if cpuDir == GraphDirectionUp {
				t = float64(len(rawLines)-1-i) / float64(len(rawLines)-1)
			} else {
				t = float64(i) / float64(len(rawLines)-1)
			}
		} else {
			t = 0.0
		}
		c := interpolateChartColor(theme.Current.Palette.ChartGradient, t)
		b.WriteString(lipgloss.NewStyle().Foreground(c).Render(line))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Memory Graph
	currentMem := s.memUsage
	peakMem := s.memUsage

	// Calculate peak only across visible points so expired spikes scroll away cleanly
	startIdxMem := max(len(s.memHistory)-totalPointsNeeded, 0)
	for _, val := range s.memHistory[startIdxMem:] {
		if val > peakMem {
			peakMem = val
		}
	}
	if len(s.memHistory) > 0 {
		currentMem = s.memHistory[len(s.memHistory)-1]
	}

	var displayLimit float64
	var memPercent float64
	var isDynamic bool

	// Check if container has a limit explicitly set
	hasLimit := s.hasMemLimit
	effectiveMemLimit := s.memLimit
	if !hasLimit && selected.Details != nil && selected.Details.HostConfig != nil && selected.Details.HostConfig.Memory > 0 {
		hasLimit = true
		if effectiveMemLimit <= 0 {
			effectiveMemLimit = float64(selected.Details.HostConfig.Memory)
		}
	}

	if hasLimit && effectiveMemLimit > 0 {
		displayLimit = effectiveMemLimit
		memPercent = (currentMem / displayLimit) * 100.0
	} else {
		isDynamic = true
		switch {
		case peakMem <= 4*units.MiB:
			displayLimit = 8 * units.MiB
		case peakMem <= 8*units.MiB:
			displayLimit = 16 * units.MiB
		case peakMem <= 16*units.MiB:
			displayLimit = 32 * units.MiB
		case peakMem <= 32*units.MiB:
			displayLimit = 64 * units.MiB
		case peakMem <= 64*units.MiB:
			displayLimit = 128 * units.MiB
		case peakMem <= 128*units.MiB:
			displayLimit = 256 * units.MiB
		case peakMem <= 256*units.MiB:
			displayLimit = 512 * units.MiB
		case peakMem <= 512*units.MiB:
			displayLimit = 1024 * units.MiB
		case peakMem <= 1024*units.MiB:
			displayLimit = 2048 * units.MiB
		case peakMem <= 2048*units.MiB:
			displayLimit = 4096 * units.MiB
		default:
			displayLimit = math.Ceil(peakMem/units.GiB) * units.GiB
		}
		if displayLimit > 0 {
			memPercent = (currentMem / displayLimit) * 100.0
		}
	}

	memDirSymbol := "▼"
	if memDir == GraphDirectionUp {
		memDirSymbol = "▲"
	}

	var memTitle string
	if isDynamic {
		memTitle = fmt.Sprintf("Memory: %s [Scale: %s | Peak: %s] [%s 'b']",
			units.BytesSize(currentMem), units.BytesSize(displayLimit),
			units.BytesSize(peakMem), memDirSymbol)
	} else {
		memTitle = fmt.Sprintf("Memory: %s / %s (%.1f%%) [Peak: %s] [%s 'b']",
			units.BytesSize(currentMem), units.BytesSize(displayLimit), memPercent,
			units.BytesSize(peakMem), memDirSymbol)
	}

	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(theme.Current.Palette.Warning).Render(memTitle))
	b.WriteString("\n")

	rawMemLines := renderBtopBraille(s.memHistory, graphWidth, graphHeight, displayLimit, memDir)
	for i, line := range rawMemLines {
		var t float64
		if len(rawMemLines) > 1 {
			if memDir == GraphDirectionUp {
				t = float64(len(rawMemLines)-1-i) / float64(len(rawMemLines)-1)
			} else {
				t = float64(i) / float64(len(rawMemLines)-1)
			}
		}
		c := interpolateChartColor(theme.Current.Palette.ChartGradient, t)
		b.WriteString(lipgloss.NewStyle().Foreground(c).Render(line))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Meta info
	labelStyle := theme.Current.Key
	valStyle := theme.Current.Value

	netText := fmt.Sprintf("%s ↓ / %s ↑", units.HumanSize(s.netRx), units.HumanSize(s.netTx))
	b.WriteString(labelStyle.Render("Net I/O:   "))
	b.WriteString(valStyle.Render(netText))
	b.WriteString("\n")

	blockText := fmt.Sprintf("%s R / %s W", units.HumanSize(float64(s.blockRead)), units.HumanSize(float64(s.blockWrite)))
	b.WriteString(labelStyle.Render("Block I/O: "))
	b.WriteString(valStyle.Render(blockText))
	b.WriteString("\n")

	b.WriteString(labelStyle.Render("Image:     "))
	b.WriteString(valStyle.Render(selected.Image))
	b.WriteString("\n")
	if selected.Ports != "" {
		b.WriteString(labelStyle.Render("Ports:     "))
		b.WriteString(valStyle.Render(selected.Ports))
		b.WriteString("\n")
	}
	if selected.IsCompose {
		b.WriteString(labelStyle.Render("Compose:   "))
		b.WriteString(valStyle.Render(selected.ProjectName + " / " + selected.Service))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	// Status feedback when action is executing
	if actionLoading != "" {
		b.WriteString(labelStyle.Render("Status:  "))
		b.WriteString(actionLoading)
		b.WriteString("\n")
	}

	return b.String()
}

func renderBtopBraille(data []float64, width, height int, scaleMax float64, dir GraphDirection) []string {
	totalPointsNeeded := width * 2
	normData := make([]float64, totalPointsNeeded)
	offset := len(data) - totalPointsNeeded

	for i := range totalPointsNeeded {
		idx := offset + i
		if idx >= 0 && idx < len(data) {
			normData[i] = math.Max(0.0, math.Min(scaleMax, data[idx]))
		} else {
			normData[i] = -1.0 // Unfilled padding before data starts (blank)
		}
	}

	mod := 0.1
	if height == 1 {
		mod = 0.3
	}

	lines := make([]string, height)
	for y := range height {
		var rowBuilder strings.Builder
		var curLow, curHigh float64
		var isBaseLine bool

		if dir == GraphDirectionUp {
			curHigh = scaleMax * float64(height-y) / float64(height)
			curLow = scaleMax * float64(height-(y+1)) / float64(height)
			isBaseLine = (y == height-1)
		} else {
			curLow = scaleMax * float64(y) / float64(height)
			curHigh = scaleMax * float64(y+1) / float64(height)
			isBaseLine = (y == 0)
		}

		for i := 0; i < totalPointsNeeded; i += 2 {
			dotsLeft := calcDots(normData[i], curLow, curHigh, isBaseLine, mod)
			dotsRight := calcDots(normData[i+1], curLow, curHigh, isBaseLine, mod)
			if dir == GraphDirectionUp {
				rowBuilder.WriteString(brailleUpSymbols[(dotsLeft*5)+dotsRight])
			} else {
				rowBuilder.WriteString(brailleDownSymbols[(dotsLeft*5)+dotsRight])
			}
		}
		lines[y] = rowBuilder.String()
	}
	return lines
}

func calcDots(val, curLow, curHigh float64, isBaseLine bool, mod float64) int {
	if val < 0.0 {
		return 0 // padding -> empty space
	}
	minDots := 0
	if isBaseLine {
		minDots = 1 // no_zero: always maintain 1 baseline dot for valid samples
	}
	if val >= curHigh {
		return 4
	}
	if val <= curLow {
		return minDots
	}
	d := int(math.Round((val-curLow)*4.0/(curHigh-curLow) + mod))
	if d < minDots {
		return minDots
	}
	if d > 4 {
		return 4
	}
	return d
}

func interpolateChartColor(stops []color.Color, t float64) color.Color {
	if len(stops) == 0 {
		return theme.Current.Palette.Accent
	}
	if len(stops) == 1 || t <= 0.0 {
		return stops[0]
	}
	if t >= 1.0 {
		return stops[len(stops)-1]
	}

	pos := t * float64(len(stops)-1)
	idx := int(pos)
	if idx >= len(stops)-1 {
		return stops[len(stops)-1]
	}

	frac := pos - float64(idx)
	r1, g1, b1, _ := stops[idx].RGBA()
	r2, g2, b2, _ := stops[idx+1].RGBA()

	return color.RGBA{
		R: uint8(math.Round(float64(r1>>8)*(1.0-frac) + float64(r2>>8)*frac)),
		G: uint8(math.Round(float64(g1>>8)*(1.0-frac) + float64(g2>>8)*frac)),
		B: uint8(math.Round(float64(b1>>8)*(1.0-frac) + float64(b2>>8)*frac)),
		A: 255,
	}
}

func RenderStatusBadge(state string) string {
	switch strings.ToLower(state) {
	case string(docker.StateRunning):
		return theme.Current.StatusRunning.Render("● RUNNING")
	case string(docker.StatePaused):
		return theme.Current.StatusPaused.Render("○ PAUSED")
	case string(docker.StateRestarting):
		return theme.Current.StatusPaused.Render("↻ RESTARTS")
	default:
		return theme.Current.StatusStopped.Render("○ " + strings.ToUpper(state))
	}
}
