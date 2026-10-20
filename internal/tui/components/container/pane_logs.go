package container

import (
	"fmt"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/xoaiPro235/docktop/internal/tui/keys"
	"github.com/xoaiPro235/docktop/internal/tui/theme"
)

type LogsPane struct {
	viewport viewport.Model
	logs     []string
	count    int // number of valid entries currently stored
	maxLines int

	autoScroll bool
	width      int
	height     int
	ready      bool
}

func NewLogsPane() LogsPane {
	return LogsPane{
		logs:       make([]string, 0, 1000),
		maxLines:   1000,
		autoScroll: true,
	}
}

func (p *LogsPane) SetSize(w, h int) {
	p.width = w
	p.height = h
	if !p.ready {
		p.viewport = viewport.New(viewport.WithWidth(w), viewport.WithHeight(h))
		p.viewport.KeyMap.PageUp = key.NewBinding(
			key.WithKeys("pgup", "pageup", "b"),
			key.WithHelp("pgup", "page up"),
		)
		p.viewport.KeyMap.PageDown = key.NewBinding(
			key.WithKeys("pgdown", "pagedown", "f", " "),
			key.WithHelp("pgdn", "page down"),
		)
		p.ready = true
	} else {
		p.viewport.SetWidth(w)
		p.viewport.SetHeight(h)
	}
	p.setContent()
}

// AppendLines adds a batch of log lines and updates the viewport once.
func (p *LogsPane) AppendLines(lines []string) {
	if len(lines) == 0 {
		return
	}
	for _, line := range lines {
		if len(p.logs) < p.maxLines {
			p.logs = append(p.logs, line)
		} else {
			copy(p.logs, p.logs[1:])
			p.logs[len(p.logs)-1] = line
		}
	}
	p.count = len(p.logs)
	if p.ready {
		p.setContent()
	}
}

// AppendLine adds a single log line.
func (p *LogsPane) AppendLine(line string) {
	p.AppendLines([]string{line})
}

func (p *LogsPane) Reset() {
	p.logs = p.logs[:0]
	p.count = 0
	p.autoScroll = true
	if p.ready {
		p.setContent()
	}
}

func (p *LogsPane) ToggleWrap() {
	p.viewport.SoftWrap = !p.viewport.SoftWrap
	if p.autoScroll {
		p.viewport.GotoBottom()
	}
}

func (p LogsPane) Update(msg tea.Msg) (LogsPane, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch {
		case key.Matches(msg, keys.ContainerKeys.ToggleWrap):
			p.ToggleWrap()
			return p, nil

		case key.Matches(msg, keys.ContainerKeys.FollowLogs):
			p.autoScroll = true
			p.viewport.GotoBottom()
			return p, nil

		case key.Matches(msg, keys.ContainerKeys.TopLogs):
			p.autoScroll = false
			p.viewport.GotoTop()
			return p, nil

		case key.Matches(msg, keys.ContainerKeys.Up, p.viewport.KeyMap.PageUp, p.viewport.KeyMap.HalfPageUp):
			p.autoScroll = false
		}

	case tea.MouseWheelMsg:
		if msg.Button == tea.MouseWheelUp {
			p.autoScroll = false
		}
	}

	p.viewport, cmd = p.viewport.Update(msg)

	// If scrolled all the way back down to the bottom, resume auto-scroll
	if p.viewport.AtBottom() {
		p.autoScroll = true
	}

	return p, cmd
}

// setContent rebuilds the viewport content using SetContentLines directly.
func (p *LogsPane) setContent() {
	if !p.ready {
		return
	}
	if len(p.logs) == 0 {
		emptyMsg := theme.Current.MutedText.Padding(1, 2).Render("Waiting for container logs... (no output produced yet)")
		p.viewport.SetContent(emptyMsg)
		return
	}

	p.viewport.SetContentLines(p.logs)
	if p.autoScroll {
		p.viewport.GotoBottom()
	}
}

func (p LogsPane) Render() string {
	if !p.ready {
		return "Initializing log viewer..."
	}

	var status string
	if p.count == 0 {
		status = theme.Current.MutedText.Render("○ WAITING")
	} else if p.autoScroll {
		status = theme.Current.StatusRunning.Render("● AUTO-SCROLL")
	} else {
		scrollPct := fmt.Sprintf("%3.f%%", p.viewport.ScrollPercent()*100)
		status = theme.Current.StatusPaused.Render("○ PAUSED ("+scrollPct+")") + " " + theme.Current.FooterKey.Render("[G]") + " " + theme.Current.Title.Render("Follow")
	}

	lineInfo := theme.Current.Key.Render(" Lines: ") + theme.Current.AccentText.Render(fmt.Sprintf("%d", p.count))
	sep := theme.Current.LineNumber.Render(" │ ")

	footer := theme.Current.Footer.
		Width(p.width).
		MaxHeight(1).
		Render(lineInfo + sep + status)

	return p.viewport.View() + "\n" + footer
}
