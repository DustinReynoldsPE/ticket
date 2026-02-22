// Package mcp provides an MCP server for AI agent access to tickets.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/EnderRealm/ticket/pkg/ticket"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewServer creates an MCP server with all ticket management tools registered.
func NewServer(ticketsDir string) *mcp.Server {
	store := ticket.NewFileStore(ticketsDir)

	server := mcp.NewServer(
		&mcp.Implementation{Name: "tk", Version: "0.1.0"},
		nil,
	)

	registerList(server, store)
	registerShow(server, store)
	registerCreate(server, store)
	registerEdit(server, store)
	registerAddNote(server, store)
	registerDep(server, store)
	registerLink(server, store)
	registerReady(server, store)
	registerBlocked(server, store)
	registerWorkflow(server)

	return server
}

// JSON representation of a ticket for MCP responses.
type ticketJSON struct {
	ID          string   `json:"id"`
	Status      string   `json:"status"`
	Deps        []string `json:"deps"`
	Links       []string `json:"links"`
	Created     string   `json:"created"`
	Type        string   `json:"type"`
	Priority    int      `json:"priority"`
	Assignee    string   `json:"assignee,omitempty"`
	ExternalRef string   `json:"external_ref,omitempty"`
	Parent      string   `json:"parent,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Design      string   `json:"design,omitempty"`
	Acceptance  string   `json:"acceptance_criteria,omitempty"`
	Notes       []noteJSON `json:"notes,omitempty"`
}

type noteJSON struct {
	Timestamp string `json:"timestamp"`
	Text      string `json:"text"`
}

func toJSON(t *ticket.Ticket) ticketJSON {
	j := ticketJSON{
		ID:          t.ID,
		Status:      string(t.Status),
		Deps:        nonNil(t.Deps),
		Links:       nonNil(t.Links),
		Created:     t.Created.UTC().Format("2006-01-02T15:04:05Z"),
		Type:        string(t.Type),
		Priority:    t.Priority,
		Assignee:    t.Assignee,
		ExternalRef: t.ExternalRef,
		Parent:      t.Parent,
		Tags:        t.Tags,
		Title:       t.Title,
	}

	// Extract body sections.
	body := t.Body
	if body != "" {
		j.Description, j.Design, j.Acceptance = parseSections(body)
	}

	for _, n := range t.Notes {
		j.Notes = append(j.Notes, noteJSON{
			Timestamp: n.Timestamp.UTC().Format("2006-01-02T15:04:05Z"),
			Text:      n.Text,
		})
	}

	return j
}

func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func parseSections(body string) (desc, design, acceptance string) {
	lines := strings.Split(body, "\n")
	var current *string
	var buf []string

	flush := func() {
		if current != nil {
			*current = strings.TrimSpace(strings.Join(buf, "\n"))
		}
		buf = nil
	}

	desc = ""
	current = &desc

	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "## Design"):
			flush()
			current = &design
		case strings.HasPrefix(line, "## Acceptance"):
			flush()
			current = &acceptance
		case strings.HasPrefix(line, "## "):
			// Other section - stop capturing known sections.
			flush()
			current = nil
		default:
			buf = append(buf, line)
		}
	}
	flush()
	return
}

func textResult(text string) (*mcp.CallToolResult, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}, nil
}

func jsonResult(v any) (*mcp.CallToolResult, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return textResult(string(data))
}

func errResult(format string, a ...any) (*mcp.CallToolResult, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf(format, a...)},
		},
		IsError: true,
	}, nil
}

// --- Tool registrations ---

type listArgs struct {
	Status   string `json:"status,omitempty" jsonschema:"filter by status: open, in_progress, needs_testing, closed"`
	Type     string `json:"type,omitempty" jsonschema:"filter by type: bug, feature, task, epic, chore"`
	Priority int    `json:"priority,omitempty" jsonschema:"filter by priority (0-4, -1 for no filter)"`
	Assignee string `json:"assignee,omitempty" jsonschema:"filter by assignee name"`
	Tag      string `json:"tag,omitempty" jsonschema:"filter by tag"`
	Parent   string `json:"parent,omitempty" jsonschema:"filter by parent ticket ID"`
}

func registerList(server *mcp.Server, store *ticket.FileStore) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ticket_list",
		Description: "List tickets with optional filters. Returns non-closed tickets by default.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args listArgs) (*mcp.CallToolResult, any, error) {
		tickets, err := store.List()
		if err != nil {
			r, _ := errResult("failed to list tickets: %v", err)
			return r, nil, nil
		}

		opts := ticket.DefaultListOptions()
		if args.Status != "" {
			opts.Status = ticket.Status(args.Status)
		} else {
			var filtered []*ticket.Ticket
			for _, t := range tickets {
				if t.Status != ticket.StatusClosed {
					filtered = append(filtered, t)
				}
			}
			tickets = filtered
		}
		if args.Type != "" {
			opts.Type = ticket.TicketType(args.Type)
		}
		if args.Priority >= 0 {
			opts.Priority = args.Priority
		}
		if args.Assignee != "" {
			opts.Assignee = args.Assignee
		}
		if args.Tag != "" {
			opts.Tag = args.Tag
		}
		if args.Parent != "" {
			opts.Parent = args.Parent
		}

		tickets = ticket.Filter(tickets, opts)
		ticket.SortByStatusPriorityID(tickets)

		var result []ticketJSON
		for _, t := range tickets {
			result = append(result, toJSON(t))
		}

		r, err := jsonResult(result)
		return r, nil, err
	})
}

type showArgs struct {
	ID string `json:"id" jsonschema:"ticket ID (supports partial matching)"`
}

func registerShow(server *mcp.Server, store *ticket.FileStore) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ticket_show",
		Description: "Show full details of a ticket by ID.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args showArgs) (*mcp.CallToolResult, any, error) {
		t, err := store.Get(args.ID)
		if err != nil {
			r, _ := errResult("ticket not found: %v", err)
			return r, nil, nil
		}

		r, err := jsonResult(toJSON(t))
		return r, nil, err
	})
}

type createArgs struct {
	Title       string `json:"title" jsonschema:"ticket title"`
	Description string `json:"description,omitempty" jsonschema:"description text"`
	Design      string `json:"design,omitempty" jsonschema:"design notes"`
	Acceptance  string `json:"acceptance,omitempty" jsonschema:"acceptance criteria"`
	Type        string `json:"type,omitempty" jsonschema:"ticket type: bug, feature, task, epic, chore (default: task)"`
	Priority    int    `json:"priority,omitempty" jsonschema:"priority 0-4, 0=highest (default: 2)"`
	Assignee    string `json:"assignee,omitempty" jsonschema:"assignee name"`
	Parent      string `json:"parent,omitempty" jsonschema:"parent ticket ID"`
	Tags        string `json:"tags,omitempty" jsonschema:"comma-separated tags"`
	ExternalRef string `json:"external_ref,omitempty" jsonschema:"external reference"`
}

func registerCreate(server *mcp.Server, store *ticket.FileStore) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ticket_create",
		Description: "Create a new ticket.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args createArgs) (*mcp.CallToolResult, any, error) {
		t := &ticket.Ticket{
			Title:    args.Title,
			Priority: 2,
		}

		if args.Type != "" {
			t.Type = ticket.TicketType(args.Type)
		} else {
			t.Type = ticket.TypeTask
		}
		if args.Priority > 0 || args.Priority == 0 {
			t.Priority = args.Priority
		}
		if args.Assignee != "" {
			t.Assignee = args.Assignee
		}
		if args.Parent != "" {
			t.Parent = args.Parent
		}
		if args.ExternalRef != "" {
			t.ExternalRef = args.ExternalRef
		}
		if args.Tags != "" {
			t.Tags = strings.Split(args.Tags, ",")
			for i := range t.Tags {
				t.Tags[i] = strings.TrimSpace(t.Tags[i])
			}
		}

		// Build body.
		var body strings.Builder
		if args.Description != "" {
			body.WriteString(args.Description + "\n")
		}
		if args.Design != "" {
			body.WriteString("\n## Design\n\n" + args.Design + "\n")
		}
		if args.Acceptance != "" {
			body.WriteString("\n## Acceptance Criteria\n\n" + args.Acceptance + "\n")
		}
		t.Body = body.String()

		if err := store.Create(t); err != nil {
			r, _ := errResult("failed to create ticket: %v", err)
			return r, nil, nil
		}

		r, err := jsonResult(toJSON(t))
		return r, nil, err
	})
}

type editArgs struct {
	ID          string `json:"id" jsonschema:"ticket ID"`
	Title       string `json:"title,omitempty" jsonschema:"new title"`
	Status      string `json:"status,omitempty" jsonschema:"new status: open, in_progress, needs_testing, closed"`
	Type        string `json:"type,omitempty" jsonschema:"new type"`
	Priority    *int   `json:"priority,omitempty" jsonschema:"new priority (0-4)"`
	Assignee    string `json:"assignee,omitempty" jsonschema:"new assignee"`
	Parent      string `json:"parent,omitempty" jsonschema:"new parent ticket ID"`
	Tags        string `json:"tags,omitempty" jsonschema:"comma-separated tags (replaces existing)"`
	ExternalRef string `json:"external_ref,omitempty" jsonschema:"external reference"`
	Description string `json:"description,omitempty" jsonschema:"new description text"`
	Design      string `json:"design,omitempty" jsonschema:"new design text"`
	Acceptance  string `json:"acceptance,omitempty" jsonschema:"new acceptance criteria"`
}

func registerEdit(server *mcp.Server, store *ticket.FileStore) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ticket_edit",
		Description: "Edit an existing ticket's fields.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args editArgs) (*mcp.CallToolResult, any, error) {
		t, err := store.Get(args.ID)
		if err != nil {
			r, _ := errResult("ticket not found: %v", err)
			return r, nil, nil
		}

		if args.Title != "" {
			t.Title = args.Title
		}
		if args.Status != "" {
			t.Status = ticket.Status(args.Status)
		}
		if args.Type != "" {
			t.Type = ticket.TicketType(args.Type)
		}
		if args.Priority != nil {
			t.Priority = *args.Priority
		}
		if args.Assignee != "" {
			t.Assignee = args.Assignee
		}
		if args.Parent != "" {
			t.Parent = args.Parent
		}
		if args.ExternalRef != "" {
			t.ExternalRef = args.ExternalRef
		}
		if args.Tags != "" {
			t.Tags = strings.Split(args.Tags, ",")
			for i := range t.Tags {
				t.Tags[i] = strings.TrimSpace(t.Tags[i])
			}
		}

		if err := store.Update(t); err != nil {
			r, _ := errResult("failed to update ticket: %v", err)
			return r, nil, nil
		}

		// Propagate status changes.
		if args.Status != "" {
			ticket.PropagateStatus(store, t.ID)
		}

		// Re-read to get propagated state.
		t, _ = store.Get(t.ID)
		r, err := jsonResult(toJSON(t))
		return r, nil, err
	})
}

type addNoteArgs struct {
	ID   string `json:"id" jsonschema:"ticket ID"`
	Text string `json:"text" jsonschema:"note text to append"`
}

func registerAddNote(server *mcp.Server, store *ticket.FileStore) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ticket_add_note",
		Description: "Append a timestamped note to a ticket.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args addNoteArgs) (*mcp.CallToolResult, any, error) {
		t, err := store.Get(args.ID)
		if err != nil {
			r, _ := errResult("ticket not found: %v", err)
			return r, nil, nil
		}

		t.Notes = append(t.Notes, ticket.Note{
			Timestamp: time.Now().UTC(),
			Text:      args.Text,
		})

		if err := store.Update(t); err != nil {
			r, _ := errResult("failed to update ticket: %v", err)
			return r, nil, nil
		}

		r, err := jsonResult(toJSON(t))
		return r, nil, err
	})
}

type depArgs struct {
	ID    string `json:"id" jsonschema:"ticket ID"`
	DepID string `json:"dep_id" jsonschema:"dependency ticket ID"`
	Action string `json:"action" jsonschema:"add or remove"`
}

func registerDep(server *mcp.Server, store *ticket.FileStore) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ticket_dep",
		Description: "Add or remove a dependency. The ticket (id) depends on dep_id.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args depArgs) (*mcp.CallToolResult, any, error) {
		t, err := store.Get(args.ID)
		if err != nil {
			r, _ := errResult("ticket not found: %v", err)
			return r, nil, nil
		}
		dep, err := store.Get(args.DepID)
		if err != nil {
			r, _ := errResult("dep ticket not found: %v", err)
			return r, nil, nil
		}

		switch args.Action {
		case "add":
			ticket.AddDep(t, dep.ID)
		case "remove":
			ticket.RemoveDep(t, dep.ID)
		default:
			r, _ := errResult("action must be 'add' or 'remove'")
			return r, nil, nil
		}

		if err := store.Update(t); err != nil {
			r, _ := errResult("failed to update ticket: %v", err)
			return r, nil, nil
		}

		r, err := jsonResult(toJSON(t))
		return r, nil, err
	})
}

type linkArgs struct {
	ID       string `json:"id" jsonschema:"ticket ID"`
	TargetID string `json:"target_id" jsonschema:"ticket to link/unlink"`
	Action   string `json:"action" jsonschema:"add or remove"`
}

func registerLink(server *mcp.Server, store *ticket.FileStore) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ticket_link",
		Description: "Add or remove a symmetric link between two tickets.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args linkArgs) (*mcp.CallToolResult, any, error) {
		t, err := store.Get(args.ID)
		if err != nil {
			r, _ := errResult("ticket not found: %v", err)
			return r, nil, nil
		}
		target, err := store.Get(args.TargetID)
		if err != nil {
			r, _ := errResult("target ticket not found: %v", err)
			return r, nil, nil
		}

		switch args.Action {
		case "add":
			ticket.AddLink(t, target)
		case "remove":
			ticket.RemoveLink(t, target)
		default:
			r, _ := errResult("action must be 'add' or 'remove'")
			return r, nil, nil
		}

		if err := store.Update(t); err != nil {
			r, _ := errResult("failed to update ticket: %v", err)
			return r, nil, nil
		}
		if err := store.Update(target); err != nil {
			r, _ := errResult("failed to update target ticket: %v", err)
			return r, nil, nil
		}

		r, err := jsonResult(toJSON(t))
		return r, nil, err
	})
}

type readyArgs struct {
	Assignee string `json:"assignee,omitempty" jsonschema:"filter by assignee"`
	Tag      string `json:"tag,omitempty" jsonschema:"filter by tag"`
}

func registerReady(server *mcp.Server, store *ticket.FileStore) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ticket_ready",
		Description: "List tickets that are ready to work on (all deps resolved, parent in_progress).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args readyArgs) (*mcp.CallToolResult, any, error) {
		ready, err := ticket.ReadyTickets(store)
		if err != nil {
			r, _ := errResult("failed to get ready tickets: %v", err)
			return r, nil, nil
		}

		opts := ticket.DefaultListOptions()
		if args.Assignee != "" {
			opts.Assignee = args.Assignee
		}
		if args.Tag != "" {
			opts.Tag = args.Tag
		}
		ready = ticket.Filter(ready, opts)
		ticket.SortByPriorityID(ready)

		var result []ticketJSON
		for _, t := range ready {
			result = append(result, toJSON(t))
		}

		r, err := jsonResult(result)
		return r, nil, err
	})
}

func registerBlocked(server *mcp.Server, store *ticket.FileStore) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ticket_blocked",
		Description: "List tickets that are blocked by unresolved dependencies.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args readyArgs) (*mcp.CallToolResult, any, error) {
		blocked, err := ticket.BlockedTickets(store)
		if err != nil {
			r, _ := errResult("failed to get blocked tickets: %v", err)
			return r, nil, nil
		}

		opts := ticket.DefaultListOptions()
		if args.Assignee != "" {
			opts.Assignee = args.Assignee
		}
		if args.Tag != "" {
			opts.Tag = args.Tag
		}
		blocked = ticket.Filter(blocked, opts)
		ticket.SortByPriorityID(blocked)

		var result []ticketJSON
		for _, t := range blocked {
			result = append(result, toJSON(t))
		}

		r, err := jsonResult(result)
		return r, nil, err
	})
}

type emptyArgs struct{}

func registerWorkflow(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "ticket_workflow",
		Description: "Show the ticket workflow guide (types, statuses, conventions).",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args emptyArgs) (*mcp.CallToolResult, any, error) {
		guide := `Ticket Workflow Guide

Types: feature, bug, task, epic, chore
Statuses: open → in_progress → needs_testing → closed

Conventions:
- Epics group related work; set --parent to nest tasks under epics
- Dependencies (deps) gate readiness: a ticket is "ready" when all deps are closed
- Parent gating: children only become ready when their parent is in_progress
- Status propagation: when all children close, parent auto-closes
- Priority: 0=critical, 1=high, 2=normal, 3=low, 4=backlog
- Tags: free-form labels (e.g., phase-1, frontend, backend)
- Notes: timestamped append-only log for progress updates`

		r, err := textResult(guide)
		return r, nil, err
	})
}
