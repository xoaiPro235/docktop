package container

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/xoaiPro235/docktop/internal/tui/theme"
)

type TopPane struct {
	viewport     viewport.Model
	currentID    string
	titles       []string
	processes    [][]string
	contentCache string
	err          error
	width        int
	height       int
	ready        bool
	loading      bool
}

func NewTopPane() TopPane {
	return TopPane{
		loading: true,
	}
}

func (p *TopPane) SetSize(w, h int) {
	p.width = w
	p.height = h
	if !p.ready {
		p.viewport = viewport.New(viewport.WithWidth(w), viewport.WithHeight(h))
		p.ready = true
	} else {
		p.viewport.SetWidth(w)
		p.viewport.SetHeight(h)
	}
	p.contentCache = "" // invalidate cache on resize
	if len(p.processes) > 0 {
		p.viewport.SetContent(renderProcessesTable(p.titles, p.processes, w))
	}
}

func (p *TopPane) SetLoading(containerID string) {
	// Only display loading message if switching to a DIFFERENT container or having no data yet
	if p.currentID != containerID || len(p.processes) == 0 {
		p.loading = true
		p.currentID = containerID
		p.contentCache = ""
		p.titles = nil
		p.processes = nil
		p.err = nil
		p.viewport.SetContent(theme.Current.MutedText.Padding(1).Render("Loading processes..."))
	}
}

func (p *TopPane) SetContent(containerID string, titles []string, processes [][]string, err error) {
	p.loading = false
	p.currentID = containerID
	p.titles = titles
	p.processes = processes
	p.err = err

	if err != nil {
		errStyle := theme.Current.ErrorText
		errMsg := err.Error()
		var msg string
		if strings.Contains(strings.ToLower(errMsg), "is not running") {
			msg = errStyle.Render("Container is not running.")
		} else {
			msg = errStyle.Render("Error fetching processes: " + errMsg)
		}
		if p.contentCache != msg {
			p.contentCache = msg
			p.viewport.SetContent(msg)
		}
		return
	}

	if len(processes) == 0 {
		msg := theme.Current.MutedText.Padding(1).Render("No processes found.")
		if p.contentCache != msg {
			p.contentCache = msg
			p.viewport.SetContent(msg)
		}
		return
	}

	newTable := renderProcessesTable(titles, processes, p.width)
	// If table content is unchanged, do NOT touch the viewport to eliminate terminal flicker
	if p.contentCache == newTable {
		return
	}

	p.contentCache = newTable
	prevOffset := p.viewport.YOffset()
	p.viewport.SetContent(newTable)
	if prevOffset > 0 {
		p.viewport.SetYOffset(prevOffset)
	}
}

func (p TopPane) Update(msg tea.Msg) (TopPane, tea.Cmd) {
	var cmd tea.Cmd
	p.viewport, cmd = p.viewport.Update(msg)
	return p, cmd
}

func (p TopPane) Render() string {
	if !p.ready {
		return "Initializing top viewer..."
	}

	var status string
	if p.loading {
		status = theme.Current.MutedText.Render("○ LOADING")
	} else if p.err != nil {
		if strings.Contains(strings.ToLower(p.err.Error()), "is not running") {
			status = theme.Current.StatusStopped.Render("○ STOPPED")
		} else {
			status = theme.Current.StatusStopped.Render("✕ ERROR")
		}
	} else {
		status = theme.Current.Key.Render(" Processes: ") + theme.Current.AccentText.Render(fmt.Sprintf("%d", len(p.processes)))
	}

	sep := theme.Current.LineNumber.Render(" │ ")
	scrollInfo := theme.Current.MutedText.Render(fmt.Sprintf("Scroll: %3.f%%", p.viewport.ScrollPercent()*100))

	footer := theme.Current.Footer.
		Width(p.width).
		MaxHeight(1).
		Render(status + sep + scrollInfo)

	return p.viewport.View() + "\n" + footer
}

func renderProcessesTable(titles []string, processes [][]string, totalWidth int) string {
	if len(titles) == 0 || len(processes) == 0 {
		return ""
	}

	colCount := len(titles)
	colWidths := make([]int, colCount)

	for i, t := range titles {
		colWidths[i] = len(t)
	}

	for _, proc := range processes {
		for i, val := range proc {
			if i < colCount && len(val) > colWidths[i] {
				// Don't expand the last column (CMD) too wide for column width calculation
				if i < colCount-1 {
					colWidths[i] = len(val)
				}
			}
		}
	}

	var sb strings.Builder

	// Header row
	headerStyle := theme.Current.SectionTitle
	sepStyle := theme.Current.LineNumber

	for i, t := range titles {
		if i == colCount-1 {
			sb.WriteString(headerStyle.Render(t))
		} else {
			format := fmt.Sprintf("%%-%ds  ", colWidths[i])
			sb.WriteString(headerStyle.Render(fmt.Sprintf(format, t)))
		}
	}
	sb.WriteString("\n")

	// Separator line
	var sepLine strings.Builder
	prefixWidth := 0
	for i := 0; i < colCount-1; i++ {
		sepLine.WriteString(strings.Repeat("─", colWidths[i]))
		sepLine.WriteString("  ")
		prefixWidth += colWidths[i] + 2
	}
	lastColWidth := max(totalWidth - prefixWidth, 20)
	sepLine.WriteString(strings.Repeat("─", lastColWidth))
	sb.WriteString(sepStyle.Render(sepLine.String()))
	sb.WriteString("\n")

	// Process rows
	rowStyleA := theme.Current.Title
	rowStyleB := theme.Current.Key
	dimStyle := theme.Current.Value

	for rowIdx, proc := range processes {
		style := rowStyleA
		if rowIdx%2 == 1 {
			style = rowStyleB
		}

		for i, val := range proc {
			if i >= colCount {
				break
			}
			if i == colCount-1 {
				sb.WriteString(style.Render(val))
			} else {
				format := fmt.Sprintf("%%-%ds  ", colWidths[i])
				cellStr := fmt.Sprintf(format, val)
				if i == 0 { // UID
					sb.WriteString(dimStyle.Render(cellStr))
				} else {
					sb.WriteString(style.Render(cellStr))
				}
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}
