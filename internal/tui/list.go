package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/EnderRealm/ticket/pkg/ticket"
)

var (
	headerStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("7"))
	cursorStyle = lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("237"))
	filterStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))

	priorityColors = map[int]lipgloss.Color{
		0: "1",  // red - critical
		1: "3",  // yellow - high
		2: "7",  // white - normal
		3: "8",  // gray - low
		4: "8",  // gray - backlog
	}

	typeColors = map[ticket.TicketType]lipgloss.Color{
		ticket.TypeBug:     "1", // red
		ticket.TypeFeature: "2", // green
		ticket.TypeEpic:    "5", // magenta
		ticket.TypeTask:    "4", // blue
		ticket.TypeChore:   "8", // gray
	}

	statusColors = map[ticket.Status]lipgloss.Color{
		ticket.StatusOpen:         "7", // white
		ticket.StatusInProgress:   "3", // yellow
		ticket.StatusNeedsTesting: "5", // magenta
		ticket.StatusClosed:       "8", // gray
	}

	statusTabs = []ticket.Status{
		"",                        // all (non-closed)
		ticket.StatusOpen,
		ticket.StatusInProgress,
		ticket.StatusNeedsTesting,
		ticket.StatusClosed,
	}

	statusTabLabels = []string{
		"all", "open", "in_progress", "needs_testing", "closed",
	}
)

type listModel struct {
	all          []*ticket.Ticket
	filtered     []*ticket.Ticket
	cursor       int
	offset       int
	width        int
	height       int
	filterText   string
	filterActive bool
	statusTab    int
}

func newListModel(tickets []*ticket.Ticket, w, h int) listModel {
	m := listModel{
		all:    tickets,
		width:  w,
		height: h,
	}
	m.applyFilters()
	return m
}

func (m *listModel) setSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *listModel) applyFilters() {
	var result []*ticket.Ticket
	statusFilter := statusTabs[m.statusTab]

	for _, t := range m.all {
		// Status tab filter.
		if statusFilter == "" {
			// "all" tab: show non-closed.
			if t.Status == ticket.StatusClosed {
				continue
			}
		} else if t.Status != statusFilter {
			continue
		}

		// Text filter.
		if m.filterText != "" {
			needle := strings.ToLower(m.filterText)
			if !strings.Contains(strings.ToLower(t.Title), needle) &&
				!strings.Contains(strings.ToLower(t.ID), needle) {
				continue
			}
		}

		result = append(result, t)
	}

	m.filtered = result
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
	m.clampOffset()
}

func (m *listModel) clampOffset() {
	visible := m.visibleRows()
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+visible {
		m.offset = m.cursor - visible + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

func (m listModel) visibleRows() int {
	// Reserve lines for: tabs, header, filter bar, help bar.
	rows := m.height - 4
	if rows < 1 {
		rows = 1
	}
	return rows
}

func (m listModel) update(msg tea.Msg) (listModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.filterActive {
			switch msg.String() {
			case "esc":
				m.filterActive = false
				m.filterText = ""
				m.applyFilters()
			case "enter":
				m.filterActive = false
			case "backspace":
				if len(m.filterText) > 0 {
					m.filterText = m.filterText[:len(m.filterText)-1]
					m.applyFilters()
				}
			default:
				if len(msg.String()) == 1 {
					m.filterText += msg.String()
					m.applyFilters()
				}
			}
			return m, nil
		}

		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.clampOffset()
			}
		case "down", "j":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
				m.clampOffset()
			}
		case "tab":
			m.statusTab = (m.statusTab + 1) % len(statusTabs)
			m.applyFilters()
		case "shift+tab":
			m.statusTab = (m.statusTab - 1 + len(statusTabs)) % len(statusTabs)
			m.applyFilters()
		case "/":
			m.filterActive = true
			m.filterText = ""
		case "G":
			m.cursor = max(0, len(m.filtered)-1)
			m.clampOffset()
		case "g":
			m.cursor = 0
			m.clampOffset()
		}
	}
	return m, nil
}

func (m listModel) view() string {
	var b strings.Builder

	// Status tabs.
	var tabs []string
	for i, label := range statusTabLabels {
		if i == m.statusTab {
			tabs = append(tabs, lipgloss.NewStyle().Bold(true).Underline(true).Render(label))
		} else {
			tabs = append(tabs, lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(label))
		}
	}
	b.WriteString(strings.Join(tabs, "  "))
	b.WriteString("\n")

	// Header row.
	b.WriteString(headerStyle.Render(fmt.Sprintf("%-9s %-3s %-11s %-14s %s", "ID", "P", "TYPE", "STATUS", "TITLE")))
	b.WriteString("\n")

	// Ticket rows.
	visible := m.visibleRows()
	end := m.offset + visible
	if end > len(m.filtered) {
		end = len(m.filtered)
	}

	for i := m.offset; i < end; i++ {
		t := m.filtered[i]
		line := m.renderRow(t)
		if i == m.cursor {
			// Pad to full width for highlight bar.
			padded := line
			lineLen := lipgloss.Width(padded)
			if lineLen < m.width {
				padded += strings.Repeat(" ", m.width-lineLen)
			}
			b.WriteString(cursorStyle.Render(padded))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")
	}

	// Pad remaining rows.
	for i := end - m.offset; i < visible; i++ {
		b.WriteString("\n")
	}

	// Filter / help bar.
	if m.filterActive {
		b.WriteString(filterStyle.Render("/ " + m.filterText + "█"))
	} else if m.filterText != "" {
		b.WriteString(filterStyle.Render("filter: " + m.filterText + "  (/ to edit, esc clears)"))
	} else {
		help := "↑↓/jk navigate  enter open  tab status  / filter  q quit"
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render(help))
	}

	return b.String()
}

func (m listModel) renderRow(t *ticket.Ticket) string {
	pStyle := lipgloss.NewStyle().Foreground(priorityColors[t.Priority])
	tStyle := lipgloss.NewStyle().Foreground(typeColors[t.Type])
	sStyle := lipgloss.NewStyle().Foreground(statusColors[t.Status])

	id := fmt.Sprintf("%-9s", t.ID)
	pri := pStyle.Render(fmt.Sprintf("P%d", t.Priority))
	typ := tStyle.Render(fmt.Sprintf("%-11s", t.Type))
	status := sStyle.Render(fmt.Sprintf("%-14s", t.Status))

	return fmt.Sprintf("%s %s  %s %s %s", id, pri, typ, status, t.Title)
}
