package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/EnderRealm/ticket/pkg/ticket"
)

var (
	formLabelStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("4")).Width(14)
	formActiveStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	formCursorStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	formHelpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	formTitleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
)

var ticketTypes = []ticket.TicketType{
	ticket.TypeTask,
	ticket.TypeFeature,
	ticket.TypeBug,
	ticket.TypeEpic,
	ticket.TypeChore,
}

type formField int

const (
	fieldTitle formField = iota
	fieldDescription
	fieldType
	fieldPriority
	fieldAssignee
	fieldCount
)

type formModel struct {
	fields   [fieldCount]string
	focus    formField
	typeIdx  int
	priority int
	width    int
	height   int
}

func newFormModel(w, h int) formModel {
	return formModel{
		typeIdx:  0, // task
		priority: 2,
		width:    w,
		height:   h,
	}
}

func (m *formModel) setSize(w, h int) {
	m.width = w
	m.height = h
}

func (m formModel) update(msg tea.Msg) (formModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return m, func() tea.Msg { return formCancelMsg{} }

		case "tab", "down":
			m.focus = (m.focus + 1) % fieldCount
		case "shift+tab", "up":
			m.focus = (m.focus - 1 + fieldCount) % fieldCount

		case "enter":
			if m.focus == fieldType {
				m.typeIdx = (m.typeIdx + 1) % len(ticketTypes)
			} else if m.focus == fieldPriority {
				m.priority = (m.priority + 1) % 5
			} else {
				return m, m.submit
			}

		case "left":
			if m.focus == fieldType {
				m.typeIdx = (m.typeIdx - 1 + len(ticketTypes)) % len(ticketTypes)
			} else if m.focus == fieldPriority {
				if m.priority > 0 {
					m.priority--
				}
			}
		case "right":
			if m.focus == fieldType {
				m.typeIdx = (m.typeIdx + 1) % len(ticketTypes)
			} else if m.focus == fieldPriority {
				if m.priority < 4 {
					m.priority++
				}
			}

		case "backspace":
			if m.focus == fieldTitle || m.focus == fieldDescription || m.focus == fieldAssignee {
				f := &m.fields[m.focus]
				if len(*f) > 0 {
					*f = (*f)[:len(*f)-1]
				}
			}

		default:
			if m.focus == fieldTitle || m.focus == fieldDescription || m.focus == fieldAssignee {
				if len(msg.String()) == 1 {
					m.fields[m.focus] += msg.String()
				}
			}
		}
	}
	return m, nil
}

func (m formModel) submit() tea.Msg {
	title := strings.TrimSpace(m.fields[fieldTitle])
	if title == "" {
		return nil
	}
	return formSubmitMsg{
		title:       title,
		description: strings.TrimSpace(m.fields[fieldDescription]),
		ticketType:  ticketTypes[m.typeIdx],
		priority:    m.priority,
		assignee:    strings.TrimSpace(m.fields[fieldAssignee]),
	}
}

func (m formModel) view() string {
	var b strings.Builder

	b.WriteString(formTitleStyle.Render("Create New Ticket"))
	b.WriteString("\n\n")

	labels := [fieldCount]string{"Title:", "Description:", "Type:", "Priority:", "Assignee:"}

	for i := formField(0); i < fieldCount; i++ {
		label := formLabelStyle.Render(labels[i])
		var val string

		switch i {
		case fieldType:
			var parts []string
			for j, tt := range ticketTypes {
				s := string(tt)
				if j == m.typeIdx {
					s = formCursorStyle.Render("[" + s + "]")
				}
				parts = append(parts, s)
			}
			val = strings.Join(parts, "  ")
		case fieldPriority:
			var parts []string
			for j := 0; j < 5; j++ {
				s := lipgloss.NewStyle().Foreground(priorityColors[j]).Render(
					strings.Repeat("P", 1) + string(rune('0'+j)),
				)
				if j == m.priority {
					s = formCursorStyle.Render("[" + s + "]")
				}
				parts = append(parts, s)
			}
			val = strings.Join(parts, "  ")
		default:
			val = m.fields[i]
		}

		cursor := "  "
		if i == m.focus {
			cursor = formCursorStyle.Render("> ")
			if i == fieldTitle || i == fieldDescription || i == fieldAssignee {
				val = formActiveStyle.Render(val + "█")
			}
		}

		b.WriteString(cursor + label + " " + val + "\n")
	}

	b.WriteString("\n")
	help := "tab/↑↓ fields  enter submit/cycle  ←→ cycle  esc cancel"
	b.WriteString(formHelpStyle.Render(help))

	return b.String()
}

// Messages
type formSubmitMsg struct {
	title       string
	description string
	ticketType  ticket.TicketType
	priority    int
	assignee    string
}

type formCancelMsg struct{}
