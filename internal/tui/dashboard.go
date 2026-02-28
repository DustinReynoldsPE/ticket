package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/EnderRealm/ticket/pkg/ticket"
)

var (
	paneHeaderStyle = lipgloss.NewStyle().Bold(true).Underline(true)
	paneActiveStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	paneDimStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	detailStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	dashHelpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	rowSelected     = lipgloss.NewStyle().Bold(true).Background(lipgloss.Color("237"))
)

type dashboardModel struct {
	all          []*ticket.Ticket
	triage       []ticket.InboxItem
	inbox        []ticket.InboxItem
	focusPane    int // 0=triage, 1=inbox
	triageCursor int
	inboxCursor  int
	triageOffset int
	inboxOffset  int
	width        int
	height       int
	filterText   string
	filterActive bool
	typeFilter   ticket.TicketType
}

func newDashboardModel(tickets []*ticket.Ticket, w, h int) dashboardModel {
	m := dashboardModel{
		all:    tickets,
		width:  w,
		height: h,
	}
	m.buildPanes()
	return m
}

func (m *dashboardModel) setSize(w, h int) {
	m.width = w
	m.height = h
}

func (m *dashboardModel) buildPanes() {
	m.triage = nil
	m.inbox = nil
	needle := strings.ToLower(m.filterText)

	for _, t := range m.all {
		if t.Stage == "" || t.Stage == ticket.StageDone {
			continue
		}
		if m.typeFilter != "" && t.Type != m.typeFilter {
			continue
		}
		if needle != "" {
			if !strings.Contains(strings.ToLower(t.Title), needle) &&
				!strings.Contains(strings.ToLower(t.ID), needle) {
				continue
			}
		}
		item := ticket.NextAction(t)
		switch item.Action {
		case ticket.ActionHumanInput:
			m.triage = append(m.triage, item)
		case ticket.ActionHumanReview:
			m.inbox = append(m.inbox, item)
		}
	}

	// Clamp cursors.
	if m.triageCursor >= len(m.triage) {
		m.triageCursor = max(0, len(m.triage)-1)
	}
	if m.inboxCursor >= len(m.inbox) {
		m.inboxCursor = max(0, len(m.inbox)-1)
	}
}

func (m dashboardModel) selected() *ticket.Ticket {
	if m.focusPane == 0 && m.triageCursor >= 0 && m.triageCursor < len(m.triage) {
		return m.triage[m.triageCursor].Ticket
	}
	if m.focusPane == 1 && m.inboxCursor >= 0 && m.inboxCursor < len(m.inbox) {
		return m.inbox[m.inboxCursor].Ticket
	}
	return nil
}

func (m dashboardModel) inputActive() bool {
	return m.filterActive
}

func (m dashboardModel) update(msg tea.Msg) (dashboardModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.filterActive {
			switch msg.String() {
			case "esc":
				m.filterActive = false
				m.filterText = ""
				m.buildPanes()
			case "enter":
				m.filterActive = false
			case "backspace":
				if len(m.filterText) > 0 {
					m.filterText = m.filterText[:len(m.filterText)-1]
					m.buildPanes()
				}
			default:
				if len(msg.String()) == 1 {
					m.filterText += msg.String()
					m.buildPanes()
				}
			}
			return m, nil
		}

		switch msg.String() {
		case "tab":
			m.focusPane = (m.focusPane + 1) % 2
		case "up", "k":
			if m.focusPane == 0 && m.triageCursor > 0 {
				m.triageCursor--
				m.clampOffset(0)
			} else if m.focusPane == 1 && m.inboxCursor > 0 {
				m.inboxCursor--
				m.clampOffset(1)
			}
		case "down", "j":
			if m.focusPane == 0 && m.triageCursor < len(m.triage)-1 {
				m.triageCursor++
				m.clampOffset(0)
			} else if m.focusPane == 1 && m.inboxCursor < len(m.inbox)-1 {
				m.inboxCursor++
				m.clampOffset(1)
			}
		case "g":
			if m.focusPane == 0 {
				m.triageCursor = 0
				m.clampOffset(0)
			} else {
				m.inboxCursor = 0
				m.clampOffset(1)
			}
		case "G":
			if m.focusPane == 0 {
				m.triageCursor = max(0, len(m.triage)-1)
				m.clampOffset(0)
			} else {
				m.inboxCursor = max(0, len(m.inbox)-1)
				m.clampOffset(1)
			}
		case "t":
			types := []ticket.TicketType{"", ticket.TypeFeature, ticket.TypeBug, ticket.TypeTask, ticket.TypeEpic, ticket.TypeChore}
			for i, tt := range types {
				if tt == m.typeFilter {
					m.typeFilter = types[(i+1)%len(types)]
					break
				}
			}
			m.buildPanes()
		case "/":
			m.filterActive = true
			m.filterText = ""
		case "esc":
			if m.filterText != "" {
				m.filterText = ""
				m.buildPanes()
			}
		}
	}
	return m, nil
}

func (m *dashboardModel) clampOffset(pane int) {
	visible := m.visibleRows()
	if pane == 0 {
		if m.triageCursor < m.triageOffset {
			m.triageOffset = m.triageCursor
		}
		if m.triageCursor >= m.triageOffset+visible {
			m.triageOffset = m.triageCursor - visible + 1
		}
	} else {
		if m.inboxCursor < m.inboxOffset {
			m.inboxOffset = m.inboxCursor
		}
		if m.inboxCursor >= m.inboxOffset+visible {
			m.inboxOffset = m.inboxCursor - visible + 1
		}
	}
}

func (m dashboardModel) visibleRows() int {
	// Reserve: 1 filter bar, 1 header, 1 help bar.
	rows := m.height - 3
	if rows < 1 {
		rows = 1
	}
	return rows
}

func (m dashboardModel) view() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	paneWidth := m.width / 2
	if paneWidth < 20 {
		paneWidth = 20
	}
	visible := m.visibleRows()

	leftPane := m.renderPane("TRIAGE", m.triage, m.triageCursor, m.triageOffset, m.focusPane == 0, paneWidth, visible)
	rightPane := m.renderPane("INBOX", m.inbox, m.inboxCursor, m.inboxOffset, m.focusPane == 1, paneWidth, visible)

	// Join side by side.
	leftLines := strings.Split(leftPane, "\n")
	rightLines := strings.Split(rightPane, "\n")
	maxLines := len(leftLines)
	if len(rightLines) > maxLines {
		maxLines = len(rightLines)
	}

	var rows []string
	for i := 0; i < maxLines; i++ {
		left := ""
		right := ""
		if i < len(leftLines) {
			left = leftLines[i]
		}
		if i < len(rightLines) {
			right = rightLines[i]
		}
		// Pad left to pane width.
		leftW := lipgloss.Width(left)
		if leftW < paneWidth {
			left += strings.Repeat(" ", paneWidth-leftW)
		}
		rows = append(rows, left+right)
	}

	content := strings.Join(rows, "\n")

	// Filter bar.
	var filterLine string
	if m.filterActive {
		filterLine = filterStyle.Render("/ " + m.filterText + "█")
	} else if m.filterText != "" {
		filterLine = filterStyle.Render("filter: " + m.filterText + "  (/ to edit, esc clears)")
	} else {
		var parts []string
		if m.typeFilter != "" {
			parts = append(parts, fmt.Sprintf("type: %s", m.typeFilter))
		}
		parts = append(parts, "(t type, / search, P pipeline)")
		filterLine = filterStyle.Render(strings.Join(parts, "  "))
	}

	// Help bar.
	help := dashHelpStyle.Render("tab pane  ↑↓/jk nav  enter open  p priority  c create  P pipeline  q quit")

	return content + "\n" + filterLine + "\n" + help
}

func (m dashboardModel) renderPane(title string, items []ticket.InboxItem, cursor, offset int, focused bool, width, visible int) string {
	var b strings.Builder

	// Header.
	headerText := fmt.Sprintf(" %s (%d)", title, len(items))
	if focused {
		b.WriteString(paneHeaderStyle.Render(paneActiveStyle.Render(headerText)))
	} else {
		b.WriteString(paneHeaderStyle.Render(paneDimStyle.Render(headerText)))
	}
	b.WriteString("\n")

	// Items.
	end := offset + visible
	if end > len(items) {
		end = len(items)
	}

	for i := offset; i < end; i++ {
		item := items[i]
		line := m.renderItem(item, width)
		if focused && i == cursor {
			padded := line
			lineLen := lipgloss.Width(padded)
			if lineLen < width {
				padded += strings.Repeat(" ", width-lineLen)
			}
			b.WriteString(rowSelected.Render(padded))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")
	}

	// Pad empty rows.
	for i := end - offset; i < visible; i++ {
		b.WriteString("\n")
	}

	return b.String()
}

func (m dashboardModel) renderItem(item ticket.InboxItem, width int) string {
	t := item.Ticket
	pStyle := lipgloss.NewStyle().Foreground(priorityColors[t.Priority])
	tStyle := lipgloss.NewStyle().Foreground(typeColors[t.Type])

	pri := pStyle.Render(fmt.Sprintf("P%d", t.Priority))
	typ := tStyle.Render(fmt.Sprintf("%-5s", shortType(t.Type)))

	// Review indicator.
	var rev string
	if t.Review == ticket.ReviewPending {
		rev = lipgloss.NewStyle().Foreground(reviewColors[t.Review]).Render(" ●")
	}

	line := fmt.Sprintf("%s %s %s%s", pri, typ, t.ID, rev)

	// Detail text (action needed).
	detail := " " + detailStyle.Render(item.Detail)

	// Title — use remaining space.
	usedWidth := lipgloss.Width(line) + 1
	titleSpace := width - usedWidth - lipgloss.Width(detail) - 1
	title := t.Title
	if titleSpace > 3 && len(title) > 0 {
		if len(title) > titleSpace {
			title = title[:titleSpace-1] + "…"
		}
		line += " " + title
	}
	line += detail

	return line
}
