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
