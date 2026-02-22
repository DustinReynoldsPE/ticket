package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/EnderRealm/ticket/pkg/ticket"
)

var (
	fieldKeyStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("4"))
	fieldValStyle   = lipgloss.NewStyle()
	sectionStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	titleStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("7"))
	detailHelpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

type detailModel struct {
	ticket *ticket.Ticket
	lines  []string
	offset int
	width  int
	height int
}

func newDetailModel(t *ticket.Ticket, w, h int) detailModel {
	m := detailModel{
		ticket: t,
		width:  w,
		height: h,
	}
	m.lines = m.render()
	return m
}

func (m *detailModel) setSize(w, h int) {
	m.width = w
	m.height = h
}

func (m detailModel) visibleRows() int {
	rows := m.height - 1 // help bar
	if rows < 1 {
		rows = 1
	}
	return rows
}

func (m detailModel) update(msg tea.Msg) (detailModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		maxOffset := max(0, len(m.lines)-m.visibleRows())
		switch msg.String() {
		case "up", "k":
			if m.offset > 0 {
				m.offset--
			}
		case "down", "j":
			if m.offset < maxOffset {
				m.offset++
			}
		case "pgup", "b":
			m.offset -= m.visibleRows()
			if m.offset < 0 {
				m.offset = 0
			}
		case "pgdown", "f", " ":
			m.offset += m.visibleRows()
			if m.offset > maxOffset {
				m.offset = maxOffset
			}
		case "g":
			m.offset = 0
		case "G":
			m.offset = maxOffset
		}
	}
	return m, nil
}

func (m detailModel) view() string {
	var b strings.Builder

	visible := m.visibleRows()
	end := m.offset + visible
	if end > len(m.lines) {
		end = len(m.lines)
	}

	for i := m.offset; i < end; i++ {
		b.WriteString(m.lines[i])
		b.WriteString("\n")
	}

	// Pad remaining.
	for i := end - m.offset; i < visible; i++ {
		b.WriteString("\n")
	}

	// Help bar.
	help := "↑↓/jk scroll  pgup/pgdn page  esc back  q quit"
	b.WriteString(detailHelpStyle.Render(help))

	return b.String()
}

func (m detailModel) render() []string {
	t := m.ticket
	var lines []string

	// Title.
	lines = append(lines, titleStyle.Render("# "+t.Title))
	lines = append(lines, "")

	// Frontmatter fields.
	lines = append(lines, m.field("id", t.ID))
	lines = append(lines, m.field("status", string(t.Status)))
	lines = append(lines, m.field("type", string(t.Type)))
	lines = append(lines, m.field("priority", fmt.Sprintf("P%d", t.Priority)))

	if t.Assignee != "" {
		lines = append(lines, m.field("assignee", t.Assignee))
	}
	if t.Parent != "" {
		lines = append(lines, m.field("parent", t.Parent))
	}
	if len(t.Deps) > 0 {
		lines = append(lines, m.field("deps", strings.Join(t.Deps, ", ")))
	}
	if len(t.Links) > 0 {
		lines = append(lines, m.field("links", strings.Join(t.Links, ", ")))
	}
	if len(t.Tags) > 0 {
		lines = append(lines, m.field("tags", strings.Join(t.Tags, ", ")))
	}
	if t.ExternalRef != "" {
		lines = append(lines, m.field("external-ref", t.ExternalRef))
	}
	lines = append(lines, m.field("created", t.Created.Format("2006-01-02 15:04")))

	lines = append(lines, "")

	// Body sections.
	body := t.Body
	if body != "" {
		for _, line := range strings.Split(body, "\n") {
			if strings.HasPrefix(line, "## ") {
				lines = append(lines, sectionStyle.Render(line))
			} else {
				lines = append(lines, line)
			}
		}
	}

	// Notes.
	if len(t.Notes) > 0 {
		lines = append(lines, sectionStyle.Render("## Notes"))
		lines = append(lines, "")
		for _, n := range t.Notes {
			lines = append(lines, fieldKeyStyle.Render(n.Timestamp.Format("2006-01-02 15:04:05")))
			for _, nl := range strings.Split(n.Text, "\n") {
				lines = append(lines, nl)
			}
			lines = append(lines, "")
		}
	}

	return lines
}

func (m detailModel) field(key, val string) string {
	return fieldKeyStyle.Render(key+":") + " " + fieldValStyle.Render(val)
}
