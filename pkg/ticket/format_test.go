package ticket

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const sampleTicketYAML = `---
id: t-abc1
status: open
deps: [t-dep1, t-dep2]
links: []
created: 2026-02-22T00:57:39Z
type: task
priority: 1
assignee: Steve Macbeth
parent: t-0f08
tags: [phase-1]
---
# Sample ticket

This is the description.

## Design

Some design notes here.

## Acceptance Criteria

Things must work.
`

func TestParse_BasicFields(t *testing.T) {
	tk, err := Parse(strings.NewReader(sampleTicketYAML))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if tk.ID != "t-abc1" {
		t.Errorf("ID = %q, want %q", tk.ID, "t-abc1")
	}
	if tk.Status != StatusOpen {
		t.Errorf("Status = %q, want %q", tk.Status, StatusOpen)
	}
	if tk.Type != TypeTask {
		t.Errorf("Type = %q, want %q", tk.Type, TypeTask)
	}
	if tk.Priority != 1 {
		t.Errorf("Priority = %d, want 1", tk.Priority)
	}
	if tk.Assignee != "Steve Macbeth" {
		t.Errorf("Assignee = %q, want %q", tk.Assignee, "Steve Macbeth")
	}
	if tk.Parent != "t-0f08" {
		t.Errorf("Parent = %q, want %q", tk.Parent, "t-0f08")
	}
	if len(tk.Deps) != 2 || tk.Deps[0] != "t-dep1" || tk.Deps[1] != "t-dep2" {
		t.Errorf("Deps = %v, want [t-dep1, t-dep2]", tk.Deps)
	}
	if len(tk.Links) != 0 {
		t.Errorf("Links = %v, want []", tk.Links)
	}
	if len(tk.Tags) != 1 || tk.Tags[0] != "phase-1" {
		t.Errorf("Tags = %v, want [phase-1]", tk.Tags)
	}
	if tk.Title != "Sample ticket" {
		t.Errorf("Title = %q, want %q", tk.Title, "Sample ticket")
	}
}

func TestParse_Notes(t *testing.T) {
	input := `---
id: t-note1
status: open
deps: []
links: []
created: 2026-01-01T00:00:00Z
type: task
priority: 2
---
# Ticket with notes

Description here.

## Notes

**2026-02-22T01:15:50Z**

First note text.

**2026-02-22T02:30:00Z**

Second note text.
`
	tk, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if len(tk.Notes) != 2 {
		t.Fatalf("len(Notes) = %d, want 2", len(tk.Notes))
	}
	if tk.Notes[0].Text != "First note text." {
		t.Errorf("Notes[0].Text = %q, want %q", tk.Notes[0].Text, "First note text.")
	}
	if tk.Notes[1].Text != "Second note text." {
		t.Errorf("Notes[1].Text = %q, want %q", tk.Notes[1].Text, "Second note text.")
	}
	expected := time.Date(2026, 2, 22, 1, 15, 50, 0, time.UTC)
	if !tk.Notes[0].Timestamp.Equal(expected) {
		t.Errorf("Notes[0].Timestamp = %v, want %v", tk.Notes[0].Timestamp, expected)
	}
}

func TestParse_MissingFrontmatter(t *testing.T) {
	_, err := Parse(strings.NewReader("# Just a heading\nSome text\n"))
	if err == nil {
		t.Error("expected error for missing frontmatter")
	}
}

func TestSerialize_RoundTrip(t *testing.T) {
	tk := &Ticket{
		ID:       "t-abc1",
		Status:   StatusOpen,
		Type:     TypeTask,
		Priority: 1,
		Assignee: "Steve Macbeth",
		Parent:   "t-0f08",
		Deps:     []string{"t-dep1"},
		Links:    []string{},
		Tags:     []string{"phase-1"},
		Created:  time.Date(2026, 2, 22, 0, 57, 39, 0, time.UTC),
		Title:    "Sample ticket",
		Body:     "\nThis is the description.\n",
	}

	data, err := Serialize(tk)
	if err != nil {
		t.Fatalf("Serialize: %v", err)
	}

	// Parse it back.
	parsed, err := Parse(strings.NewReader(string(data)))
	if err != nil {
		t.Fatalf("Parse after Serialize: %v", err)
	}

	if parsed.ID != tk.ID {
		t.Errorf("ID = %q, want %q", parsed.ID, tk.ID)
	}
	if parsed.Title != tk.Title {
		t.Errorf("Title = %q, want %q", parsed.Title, tk.Title)
	}
	if parsed.Status != tk.Status {
		t.Errorf("Status = %q, want %q", parsed.Status, tk.Status)
	}
	if len(parsed.Deps) != 1 || parsed.Deps[0] != "t-dep1" {
		t.Errorf("Deps = %v, want [t-dep1]", parsed.Deps)
	}
}

func TestSerialize_EmptyArrays(t *testing.T) {
	tk := &Ticket{
		ID:       "t-test",
		Status:   StatusOpen,
		Type:     TypeBug,
		Priority: 2,
		Deps:     []string{},
		Links:    []string{},
		Created:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Title:    "Test",
		Body:     "\n",
	}
	data, err := Serialize(tk)
	if err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	s := string(data)
	if !strings.Contains(s, "deps: []") {
		t.Error("empty deps should serialize as []")
	}
	if !strings.Contains(s, "links: []") {
		t.Error("empty links should serialize as []")
	}
}

func TestSerialize_WithNotes(t *testing.T) {
	tk := &Ticket{
		ID:       "t-test",
		Status:   StatusOpen,
		Type:     TypeTask,
		Priority: 2,
		Deps:     []string{},
		Links:    []string{},
		Created:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Title:    "Test",
		Body:     "\nDescription.\n",
		Notes: []Note{
			{Timestamp: time.Date(2026, 2, 22, 1, 0, 0, 0, time.UTC), Text: "A note."},
		},
	}
	data, err := Serialize(tk)
	if err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	s := string(data)
	if !strings.Contains(s, "## Notes") {
		t.Error("serialized output should contain ## Notes")
	}
	if !strings.Contains(s, "**2026-02-22T01:00:00Z**") {
		t.Error("serialized output should contain note timestamp")
	}
	if !strings.Contains(s, "A note.") {
		t.Error("serialized output should contain note text")
	}
}

func TestParse_RealTicketFiles(t *testing.T) {
	// Parse actual ticket files from the repo to verify compatibility.
	dir := filepath.Join("..", "..", ".tickets")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("no .tickets directory: %v", err)
	}

	var parsed int
	for _, e := range entries {
		if !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		f, err := os.Open(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Errorf("open %s: %v", e.Name(), err)
			continue
		}
		tk, err := Parse(f)
		f.Close()
		if err != nil {
			t.Errorf("parse %s: %v", e.Name(), err)
			continue
		}
		if tk.ID == "" {
			t.Errorf("%s: empty ID", e.Name())
		}
		if tk.Title == "" {
			t.Errorf("%s: empty Title", e.Name())
		}
		parsed++
	}
	if parsed == 0 {
		t.Error("no ticket files were parsed")
	}
	t.Logf("successfully parsed %d ticket files", parsed)
}
