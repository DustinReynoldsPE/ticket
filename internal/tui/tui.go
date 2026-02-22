// Package tui provides the interactive terminal UI for browsing and editing tickets.
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/EnderRealm/ticket/pkg/ticket"
)

type view int

const (
	viewList view = iota
	viewDetail
)

// App is the top-level bubbletea model.
type App struct {
	store    *ticket.FileStore
	tickets  []*ticket.Ticket
	list     listModel
	detail   detailModel
	current  view
	width    int
	height   int
	err      error
}

// New creates a new App rooted at the given ticket directory.
func New(ticketsDir string) App {
	store := ticket.NewFileStore(ticketsDir)
	return App{
		store:   store,
		current: viewList,
	}
}

type ticketsLoadedMsg []*ticket.Ticket
type errMsg error

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
		return a, nil

	case ticketsLoadedMsg:
		a.tickets = msg
		a.list = newListModel(a.tickets, a.width, a.height)
		return a, nil

	case errMsg:
		a.err = msg
		return a, tea.Quit

	case tea.KeyMsg:
		switch a.current {
		case viewList:
			if msg.String() == "enter" && len(a.list.filtered) > 0 {
				t := a.list.filtered[a.list.cursor]
				a.detail = newDetailModel(t, a.width, a.height)
				a.current = viewDetail
				return a, nil
			}
			if msg.String() == "q" && !a.list.filterActive {
				return a, tea.Quit
			}
		case viewDetail:
			if msg.String() == "esc" {
				a.current = viewList
				return a, nil
			}
			if msg.String() == "q" {
				return a, tea.Quit
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
	}

	return a, nil
}

func (a App) View() string {
	if a.err != nil {
		return fmt.Sprintf("Error: %v\n", a.err)
	}

	switch a.current {
	case viewDetail:
		return a.detail.view()
	default:
		return a.list.view()
	}
}
