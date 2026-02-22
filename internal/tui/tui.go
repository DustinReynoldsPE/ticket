// Package tui provides the interactive terminal UI for browsing and editing tickets.
package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/EnderRealm/ticket/pkg/ticket"
)

type view int

const (
	viewList view = iota
	viewDetail
	viewForm
)

// App is the top-level bubbletea model.
type App struct {
	store   *ticket.FileStore
	tickets []*ticket.Ticket
	list    listModel
	detail  detailModel
	form    formModel
	current view
	width   int
	height  int
	status  string // transient status message
	err     error
}

// New creates a new App rooted at the given ticket directory.
func New(ticketsDir string) App {
	store := ticket.NewFileStore(ticketsDir)
	return App{
		store:   store,
		current: viewList,
	}
}

// Messages

type ticketsLoadedMsg []*ticket.Ticket
type errMsg error
type statusMsg string
type clearStatusMsg struct{}

// cycleStatusMsg requests a status cycle on the selected ticket.
type cycleStatusMsg struct{ id string }

// cyclePriorityMsg requests a priority cycle on the selected ticket.
type cyclePriorityMsg struct{ id string }

// setAssigneeMsg sets the assignee on a ticket.
type setAssigneeMsg struct {
	id       string
	assignee string
}

// addNoteMsg adds a note to a ticket.
type addNoteMsg struct {
	id   string
	text string
}

func loadTickets(store *ticket.FileStore) tea.Cmd {
	return func() tea.Msg {
		tickets, err := store.List()
		if err != nil {
			return errMsg(err)
		}
		ticket.SortByStatusPriorityID(tickets)
		return ticketsLoadedMsg(tickets)
	}
}

func clearStatusAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(time.Time) tea.Msg {
		return clearStatusMsg{}
	})
}

func (a App) Init() tea.Cmd {
	return loadTickets(a.store)
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.list.setSize(a.width, a.height)
		a.detail.setSize(a.width, a.height)
		a.form.setSize(a.width, a.height)
		return a, nil

	case ticketsLoadedMsg:
		a.tickets = msg
		a.list = newListModel(a.tickets, a.width, a.height)
		// Refresh detail view if currently showing a ticket.
		if a.current == viewDetail && a.detail.ticket != nil {
			for _, t := range a.tickets {
				if t.ID == a.detail.ticket.ID {
					a.detail = newDetailModel(t, a.width, a.height)
					break
				}
			}
		}
		return a, nil

	case errMsg:
		a.err = msg
		return a, tea.Quit

	case statusMsg:
		a.status = string(msg)
		return a, clearStatusAfter(3 * time.Second)

	case clearStatusMsg:
		a.status = ""
		return a, nil

	// Mutation messages - handled at App level where store is accessible.
	case cycleStatusMsg:
		return a, a.handleCycleStatus(msg.id)
	case cyclePriorityMsg:
		return a, a.handleCyclePriority(msg.id)
	case setAssigneeMsg:
		return a, a.handleSetAssignee(msg.id, msg.assignee)
	case addNoteMsg:
		return a, a.handleAddNote(msg.id, msg.text)
	case formSubmitMsg:
		return a, a.handleCreateTicket(msg)
	case formCancelMsg:
		a.current = viewList
		return a, nil

	case tea.KeyMsg:
		switch a.current {
		case viewList:
			if a.list.inputActive() {
				break // let list handle it
			}
			switch msg.String() {
			case "enter":
				if len(a.list.filtered) > 0 {
					t := a.list.filtered[a.list.cursor]
					a.detail = newDetailModel(t, a.width, a.height)
					a.current = viewDetail
					return a, nil
				}
			case "q":
				return a, tea.Quit
			case "c":
				a.form = newFormModel(a.width, a.height)
				a.current = viewForm
				return a, nil
			case "s":
				if len(a.list.filtered) > 0 {
					t := a.list.filtered[a.list.cursor]
					return a, func() tea.Msg { return cycleStatusMsg{id: t.ID} }
				}
			case "p":
				if len(a.list.filtered) > 0 {
					t := a.list.filtered[a.list.cursor]
					return a, func() tea.Msg { return cyclePriorityMsg{id: t.ID} }
				}
			}

		case viewDetail:
			if a.detail.inputActive() {
				break // let detail handle its text input
			}
			switch msg.String() {
			case "esc":
				// Refresh detail ticket from store in case it changed.
				a.current = viewList
				return a, nil
			case "q":
				return a, tea.Quit
			case "s":
				return a, func() tea.Msg { return cycleStatusMsg{id: a.detail.ticket.ID} }
			case "p":
				return a, func() tea.Msg { return cyclePriorityMsg{id: a.detail.ticket.ID} }
			case "a":
				a.detail.startInput(inputAssignee)
				return a, nil
			case "n":
				a.detail.startInput(inputNote)
				return a, nil
			}
		}
	}

	// Delegate to child models.
	switch a.current {
	case viewList:
		var cmd tea.Cmd
		a.list, cmd = a.list.update(msg)
		return a, cmd
	case viewDetail:
		var cmd tea.Cmd
		a.detail, cmd = a.detail.update(msg)
		return a, cmd
	case viewForm:
		var cmd tea.Cmd
		a.form, cmd = a.form.update(msg)
		return a, cmd
	}

	return a, nil
}

func (a App) View() string {
	if a.err != nil {
		return fmt.Sprintf("Error: %v\n", a.err)
	}

	var content string
	switch a.current {
	case viewDetail:
		content = a.detail.view()
	case viewForm:
		content = a.form.view()
	default:
		content = a.list.view()
	}

	if a.status != "" {
		// Replace last line with status.
		lines := strings.Split(content, "\n")
		if len(lines) > 0 {
			lines[len(lines)-1] = a.status
		}
		content = strings.Join(lines, "\n")
	}

	return content
}

// Mutation handlers

var statusCycle = []ticket.Status{
	ticket.StatusOpen,
	ticket.StatusInProgress,
	ticket.StatusNeedsTesting,
	ticket.StatusClosed,
}

func (a *App) handleCycleStatus(id string) tea.Cmd {
	t, err := a.store.Get(id)
	if err != nil {
		return func() tea.Msg { return statusMsg("error: " + err.Error()) }
	}

	// Find next status in cycle.
	next := ticket.StatusOpen
	for i, s := range statusCycle {
		if s == t.Status {
			next = statusCycle[(i+1)%len(statusCycle)]
			break
		}
	}

	t.Status = next
	if err := a.store.Update(t); err != nil {
		return func() tea.Msg { return statusMsg("error: " + err.Error()) }
	}
	ticket.PropagateStatus(a.store, t.ID)

	msg := fmt.Sprintf("%s -> %s", id, next)
	return tea.Batch(
		loadTickets(a.store),
		func() tea.Msg { return statusMsg(msg) },
	)
}

func (a *App) handleCyclePriority(id string) tea.Cmd {
	t, err := a.store.Get(id)
	if err != nil {
		return func() tea.Msg { return statusMsg("error: " + err.Error()) }
	}

	t.Priority = (t.Priority + 1) % 5
	if err := a.store.Update(t); err != nil {
		return func() tea.Msg { return statusMsg("error: " + err.Error()) }
	}

	msg := fmt.Sprintf("%s -> P%d", id, t.Priority)
	return tea.Batch(
		loadTickets(a.store),
		func() tea.Msg { return statusMsg(msg) },
	)
}

func (a *App) handleSetAssignee(id, assignee string) tea.Cmd {
	t, err := a.store.Get(id)
	if err != nil {
		return func() tea.Msg { return statusMsg("error: " + err.Error()) }
	}

	t.Assignee = assignee
	if err := a.store.Update(t); err != nil {
		return func() tea.Msg { return statusMsg("error: " + err.Error()) }
	}

	msg := fmt.Sprintf("%s assignee -> %s", id, assignee)
	return tea.Batch(
		loadTickets(a.store),
		func() tea.Msg { return statusMsg(msg) },
	)
}

func (a *App) handleAddNote(id, text string) tea.Cmd {
	t, err := a.store.Get(id)
	if err != nil {
		return func() tea.Msg { return statusMsg("error: " + err.Error()) }
	}

	t.Notes = append(t.Notes, ticket.Note{
		Timestamp: time.Now().UTC(),
		Text:      text,
	})

	// Strip existing notes section from body to avoid duplication.
	if idx := strings.Index(t.Body, "\n## Notes\n"); idx >= 0 {
		t.Body = t.Body[:idx+1]
	} else if strings.HasPrefix(t.Body, "## Notes\n") {
		t.Body = "\n"
	}

	if err := a.store.Update(t); err != nil {
		return func() tea.Msg { return statusMsg("error: " + err.Error()) }
	}

	msg := fmt.Sprintf("Note added to %s", id)
	return tea.Batch(
		loadTickets(a.store),
		func() tea.Msg { return statusMsg(msg) },
	)
}

func (a *App) handleCreateTicket(msg formSubmitMsg) tea.Cmd {
	t := &ticket.Ticket{
		Title:    msg.title,
		Type:     msg.ticketType,
		Priority: msg.priority,
		Assignee: msg.assignee,
	}

	if msg.description != "" {
		t.Body = msg.description + "\n"
	}

	if err := a.store.Create(t); err != nil {
		return func() tea.Msg { return statusMsg("error: " + err.Error()) }
	}

	a.current = viewList
	status := fmt.Sprintf("Created %s: %s", t.ID, t.Title)
	return tea.Batch(
		loadTickets(a.store),
		func() tea.Msg { return statusMsg(status) },
	)
}
