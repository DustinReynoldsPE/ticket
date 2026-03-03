package mcp_test

import (
	"context"
	"encoding/json"
	"testing"

	ticketmcp "github.com/EnderRealm/ticket/internal/mcp"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func testServer(t *testing.T) *mcp.ClientSession {
	t.Helper()
	dir := t.TempDir()
	server := ticketmcp.NewServer(dir)

	st, ct := mcp.NewInMemoryTransports()

	ctx := context.Background()
	go server.Run(ctx, st)

	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0.1"}, nil)
	session, err := client.Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { session.Close() })
	return session
}

func TestCreateTicket(t *testing.T) {
	session := testServer(t)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "ticket_create",
		Arguments: map[string]any{
			"title":       "Test ticket from MCP",
			"type":        "task",
			"description": "Created via in-process MCP test",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("tool returned error: %v", result.Content)
	}

	// Parse the JSON response to verify fields.
	text := result.Content[0].(*mcp.TextContent).Text
	var ticket map[string]any
	if err := json.Unmarshal([]byte(text), &ticket); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}

	if ticket["id"] == nil || ticket["id"] == "" {
		t.Error("ticket ID is empty")
	}
	if ticket["title"] != "Test ticket from MCP" {
		t.Errorf("title = %q, want %q", ticket["title"], "Test ticket from MCP")
	}
	if ticket["stage"] != "triage" {
		t.Errorf("stage = %q, want %q", ticket["stage"], "triage")
	}
	if ticket["status"] != "open" {
		t.Errorf("status = %q, want %q", ticket["status"], "open")
	}
	created, _ := ticket["created"].(string)
	if created == "" || created == "0001-01-01T00:00:00Z" {
		t.Errorf("created = %q, want non-zero timestamp", created)
	}
}

func TestAddNotePreservesNewlines(t *testing.T) {
	session := testServer(t)
	ctx := context.Background()

	// Create a ticket.
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "ticket_create",
		Arguments: map[string]any{"title": "Note test", "type": "task"},
	})
	if err != nil {
		t.Fatal(err)
	}
	text := result.Content[0].(*mcp.TextContent).Text
	var created map[string]any
	json.Unmarshal([]byte(text), &created)
	id := created["id"].(string)

	// Add a note with double newlines.
	noteText := "## Triage\n\n**Risk:** low\n\n**Scope:** single task"
	result, err = session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "ticket_add_note",
		Arguments: map[string]any{"id": id, "text": noteText},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("add_note error: %v", result.Content)
	}

	// Read the ticket back and check notes.
	result, err = session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "ticket_show",
		Arguments: map[string]any{"id": id},
	})
	if err != nil {
		t.Fatal(err)
	}
	text = result.Content[0].(*mcp.TextContent).Text
	var shown map[string]any
	json.Unmarshal([]byte(text), &shown)

	notes, ok := shown["notes"].([]any)
	if !ok {
		t.Fatalf("notes is not an array: %T", shown["notes"])
	}
	if len(notes) != 1 {
		t.Errorf("expected 1 note, got %d", len(notes))
		for i, n := range notes {
			note := n.(map[string]any)
			t.Logf("  note[%d]: %q", i, note["text"])
		}
	}
	if len(notes) > 0 {
		note := notes[0].(map[string]any)
		if note["text"] != noteText {
			t.Errorf("note text = %q, want %q", note["text"], noteText)
		}
	}
}

func TestAddMultipleNotes(t *testing.T) {
	session := testServer(t)
	ctx := context.Background()

	// Create a ticket.
	result, _ := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "ticket_create",
		Arguments: map[string]any{"title": "Multi note test", "type": "task"},
	})
	text := result.Content[0].(*mcp.TextContent).Text
	var created map[string]any
	json.Unmarshal([]byte(text), &created)
	id := created["id"].(string)

	// Add first note with ## headings and blank lines.
	note1 := "## Triage\n\n**Risk:** low\n\n**Scope:** single task\n\n**Key decisions:**\n- Decision one (human)\n- Decision two (human)"
	session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "ticket_add_note",
		Arguments: map[string]any{"id": id, "text": note1},
	})

	// Add second note.
	note2 := "## Spec\n\n**Scope:**\n- In: feature A\n- Out: feature B"
	session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "ticket_add_note",
		Arguments: map[string]any{"id": id, "text": note2},
	})

	// Read back.
	result, _ = session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "ticket_show",
		Arguments: map[string]any{"id": id},
	})
	text = result.Content[0].(*mcp.TextContent).Text
	var shown map[string]any
	json.Unmarshal([]byte(text), &shown)

	notes, _ := shown["notes"].([]any)
	if len(notes) != 2 {
		t.Errorf("expected 2 notes, got %d", len(notes))
		for i, n := range notes {
			note := n.(map[string]any)
			t.Logf("  note[%d]: %q", i, note["text"])
		}
	}
	if len(notes) >= 1 {
		n := notes[0].(map[string]any)
		if n["text"] != note1 {
			t.Errorf("note[0] text mismatch\ngot:  %q\nwant: %q", n["text"], note1)
		}
	}
	if len(notes) >= 2 {
		n := notes[1].(map[string]any)
		if n["text"] != note2 {
			t.Errorf("note[1] text mismatch\ngot:  %q\nwant: %q", n["text"], note2)
		}
	}
}

func TestEditBodyFields(t *testing.T) {
	session := testServer(t)
	ctx := context.Background()

	// Create a ticket.
	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "ticket_create",
		Arguments: map[string]any{"title": "Edit body test", "type": "feature"},
	})
	if err != nil {
		t.Fatal(err)
	}
	text := result.Content[0].(*mcp.TextContent).Text
	var created map[string]any
	json.Unmarshal([]byte(text), &created)
	id := created["id"].(string)

	// Edit with description, design, and acceptance.
	result, err = session.CallTool(ctx, &mcp.CallToolParams{
		Name: "ticket_edit",
		Arguments: map[string]any{
			"id":          id,
			"description": "Updated description text",
			"design":      "The design plan",
			"acceptance":  "When X, the system shall Y",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("edit error: %v", result.Content)
	}

	// Read back and verify all three fields persisted.
	result, err = session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "ticket_show",
		Arguments: map[string]any{"id": id},
	})
	if err != nil {
		t.Fatal(err)
	}
	text = result.Content[0].(*mcp.TextContent).Text
	var shown map[string]any
	json.Unmarshal([]byte(text), &shown)

	if shown["description"] != "Updated description text" {
		t.Errorf("description = %q, want %q", shown["description"], "Updated description text")
	}
	if shown["design"] != "The design plan" {
		t.Errorf("design = %q, want %q", shown["design"], "The design plan")
	}
	if shown["acceptance_criteria"] != "When X, the system shall Y" {
		t.Errorf("acceptance = %q, want %q", shown["acceptance_criteria"], "When X, the system shall Y")
	}

	// Edit with test_results.
	result, err = session.CallTool(ctx, &mcp.CallToolParams{
		Name: "ticket_edit",
		Arguments: map[string]any{
			"id":           id,
			"test_results": "- [x] All tests pass\n- [x] No regressions",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("edit test_results error: %v", result.Content)
	}

	// Read back and verify test_results persisted alongside other fields.
	result, err = session.CallTool(ctx, &mcp.CallToolParams{
		Name:      "ticket_show",
		Arguments: map[string]any{"id": id},
	})
	if err != nil {
		t.Fatal(err)
	}
	text = result.Content[0].(*mcp.TextContent).Text
	json.Unmarshal([]byte(text), &shown)

	if shown["test_results"] != "- [x] All tests pass\n- [x] No regressions" {
		t.Errorf("test_results = %q, want %q", shown["test_results"], "- [x] All tests pass\n- [x] No regressions")
	}
	// Verify earlier fields weren't clobbered.
	if shown["description"] != "Updated description text" {
		t.Errorf("description clobbered: %q", shown["description"])
	}
	if shown["design"] != "The design plan" {
		t.Errorf("design clobbered: %q", shown["design"])
	}
}

func TestCreateTicketMissingTitle(t *testing.T) {
	session := testServer(t)

	_, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "ticket_create",
		Arguments: map[string]any{},
	})
	if err == nil {
		t.Error("expected error for missing title")
	}
}

func createTestTicket(t *testing.T, session *mcp.ClientSession) string {
	t.Helper()
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "ticket_create",
		Arguments: map[string]any{
			"title": "Claim test ticket",
			"type":  "task",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	text := result.Content[0].(*mcp.TextContent).Text
	var ticket map[string]any
	if err := json.Unmarshal([]byte(text), &ticket); err != nil {
		t.Fatal(err)
	}
	return ticket["id"].(string)
}

func TestClaimUnassigned(t *testing.T) {
	session := testServer(t)
	id := createTestTicket(t, session)

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "ticket_claim",
		Arguments: map[string]any{
			"id":       id,
			"assignee": "agent:builder",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("claim returned error: %s", result.Content[0].(*mcp.TextContent).Text)
	}

	text := result.Content[0].(*mcp.TextContent).Text
	var ticket map[string]any
	if err := json.Unmarshal([]byte(text), &ticket); err != nil {
		t.Fatal(err)
	}
	if ticket["assignee"] != "agent:builder" {
		t.Errorf("assignee = %q, want %q", ticket["assignee"], "agent:builder")
	}
}

func TestClaimAlreadyAssignedFails(t *testing.T) {
	session := testServer(t)
	id := createTestTicket(t, session)

	// First claim succeeds.
	_, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "ticket_claim",
		Arguments: map[string]any{
			"id":       id,
			"assignee": "agent:alpha",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Second claim by different agent fails.
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "ticket_claim",
		Arguments: map[string]any{
			"id":       id,
			"assignee": "agent:beta",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.IsError {
		t.Error("expected error when claiming ticket assigned to someone else")
	}
}

func TestClaimForceOverride(t *testing.T) {
	session := testServer(t)
	id := createTestTicket(t, session)

	// Claim as alpha.
	_, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "ticket_claim",
		Arguments: map[string]any{
			"id":       id,
			"assignee": "agent:alpha",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Force claim as beta.
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "ticket_claim",
		Arguments: map[string]any{
			"id":       id,
			"assignee": "agent:beta",
			"force":    true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("force claim returned error: %s", result.Content[0].(*mcp.TextContent).Text)
	}

	text := result.Content[0].(*mcp.TextContent).Text
	var ticket map[string]any
	if err := json.Unmarshal([]byte(text), &ticket); err != nil {
		t.Fatal(err)
	}
	if ticket["assignee"] != "agent:beta" {
		t.Errorf("assignee = %q, want %q", ticket["assignee"], "agent:beta")
	}
}

func TestClaimSameAssigneeNoop(t *testing.T) {
	session := testServer(t)
	id := createTestTicket(t, session)

	// Claim as alpha.
	_, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "ticket_claim",
		Arguments: map[string]any{
			"id":       id,
			"assignee": "agent:alpha",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Claim again as alpha — should succeed (no-op).
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "ticket_claim",
		Arguments: map[string]any{
			"id":       id,
			"assignee": "agent:alpha",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Errorf("re-claim by same assignee should not error: %s", result.Content[0].(*mcp.TextContent).Text)
	}
}

func TestVersionIncrements(t *testing.T) {
	session := testServer(t)
	id := createTestTicket(t, session)

	// Newly created ticket should have version 1.
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "ticket_show",
		Arguments: map[string]any{"id": id},
	})
	if err != nil {
		t.Fatal(err)
	}
	text := result.Content[0].(*mcp.TextContent).Text
	var ticket map[string]any
	if err := json.Unmarshal([]byte(text), &ticket); err != nil {
		t.Fatal(err)
	}
	if v := ticket["version"].(float64); v != 1 {
		t.Errorf("initial version = %v, want 1", v)
	}

	// Edit should increment to 2.
	result, err = session.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "ticket_edit",
		Arguments: map[string]any{
			"id":    id,
			"title": "Updated title",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError {
		t.Fatalf("edit returned error: %s", result.Content[0].(*mcp.TextContent).Text)
	}
	text = result.Content[0].(*mcp.TextContent).Text
	if err := json.Unmarshal([]byte(text), &ticket); err != nil {
		t.Fatal(err)
	}
	if v := ticket["version"].(float64); v != 2 {
		t.Errorf("version after edit = %v, want 2", v)
	}
}
