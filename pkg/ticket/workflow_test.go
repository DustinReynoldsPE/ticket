package ticket

import (
	"testing"
	"time"
)

func wfTicket(id string, status Status, parent string) *Ticket {
	return &Ticket{
		ID:       id,
		Status:   status,
		Type:     TypeTask,
		Priority: 2,
		Parent:   parent,
		Deps:     []string{},
		Links:    []string{},
		Created:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Title:    "Ticket " + id,
		Body:     "\n",
	}
}

func TestPropagateStatus_AllChildrenClosed(t *testing.T) {
	store := NewFileStore(t.TempDir())
	epic := wfTicket("t-epic", StatusInProgress, "")
	epic.Type = TypeEpic
	child1 := wfTicket("t-c1", StatusClosed, "t-epic")
	child2 := wfTicket("t-c2", StatusClosed, "t-epic")

	for _, tk := range []*Ticket{epic, child1, child2} {
		if err := store.Create(tk); err != nil {
			t.Fatal(err)
		}
	}

	changes, err := PropagateStatus(store, "t-c2")
	if err != nil {
		t.Fatalf("PropagateStatus: %v", err)
	}
	if len(changes) != 1 {
		t.Fatalf("len(changes) = %d, want 1", len(changes))
	}
	if changes[0].NewStatus != StatusClosed {
		t.Errorf("new status = %q, want closed", changes[0].NewStatus)
	}

	// Verify parent was updated on disk.
	parent, _ := store.Get("t-epic")
	if parent.Status != StatusClosed {
		t.Errorf("parent status = %q, want closed", parent.Status)
	}
}

func TestPropagateStatus_AllNeedsTesting(t *testing.T) {
	store := NewFileStore(t.TempDir())
	epic := wfTicket("t-epic", StatusInProgress, "")
	epic.Type = TypeEpic
	child1 := wfTicket("t-c1", StatusNeedsTesting, "t-epic")
	child2 := wfTicket("t-c2", StatusClosed, "t-epic")

	for _, tk := range []*Ticket{epic, child1, child2} {
		if err := store.Create(tk); err != nil {
			t.Fatal(err)
		}
	}

	changes, err := PropagateStatus(store, "t-c1")
	if err != nil {
		t.Fatalf("PropagateStatus: %v", err)
	}
	if len(changes) != 1 {
		t.Fatalf("len(changes) = %d, want 1", len(changes))
	}
	if changes[0].NewStatus != StatusNeedsTesting {
		t.Errorf("new status = %q, want needs_testing", changes[0].NewStatus)
	}
}

func TestPropagateStatus_MixedNoChange(t *testing.T) {
	store := NewFileStore(t.TempDir())
	epic := wfTicket("t-epic", StatusInProgress, "")
	epic.Type = TypeEpic
	child1 := wfTicket("t-c1", StatusClosed, "t-epic")
	child2 := wfTicket("t-c2", StatusOpen, "t-epic")

	for _, tk := range []*Ticket{epic, child1, child2} {
		if err := store.Create(tk); err != nil {
			t.Fatal(err)
		}
	}

	changes, err := PropagateStatus(store, "t-c1")
	if err != nil {
		t.Fatalf("PropagateStatus: %v", err)
	}
	if len(changes) != 0 {
		t.Errorf("expected no changes for mixed status, got %d", len(changes))
	}
}

func TestPropagateStatus_NoParent(t *testing.T) {
	store := NewFileStore(t.TempDir())
	tk := wfTicket("t-1", StatusClosed, "")
	if err := store.Create(tk); err != nil {
		t.Fatal(err)
	}

	changes, err := PropagateStatus(store, "t-1")
	if err != nil {
		t.Fatalf("PropagateStatus: %v", err)
	}
	if len(changes) != 0 {
		t.Errorf("expected no changes for orphan ticket, got %d", len(changes))
	}
}

func TestPropagateStatus_Recursive(t *testing.T) {
	store := NewFileStore(t.TempDir())
	grandparent := wfTicket("t-gp", StatusInProgress, "")
	grandparent.Type = TypeEpic
	parent := wfTicket("t-p", StatusInProgress, "t-gp")
	parent.Type = TypeEpic
	child := wfTicket("t-c", StatusClosed, "t-p")

	for _, tk := range []*Ticket{grandparent, parent, child} {
		if err := store.Create(tk); err != nil {
			t.Fatal(err)
		}
	}

	// t-c is only child of t-p → t-p should close.
	// t-p is only child of t-gp → t-gp should close too.
	changes, err := PropagateStatus(store, "t-c")
	if err != nil {
		t.Fatalf("PropagateStatus: %v", err)
	}
	if len(changes) != 2 {
		t.Fatalf("len(changes) = %d, want 2", len(changes))
	}

	gp, _ := store.Get("t-gp")
	if gp.Status != StatusClosed {
		t.Errorf("grandparent status = %q, want closed", gp.Status)
	}
}

func TestSetStatus(t *testing.T) {
	store := NewFileStore(t.TempDir())
	epic := wfTicket("t-epic", StatusInProgress, "")
	epic.Type = TypeEpic
	child := wfTicket("t-c1", StatusOpen, "t-epic")

	for _, tk := range []*Ticket{epic, child} {
		if err := store.Create(tk); err != nil {
			t.Fatal(err)
		}
	}

	changes, err := SetStatus(store, "t-c1", StatusClosed)
	if err != nil {
		t.Fatalf("SetStatus: %v", err)
	}

	// Child closed → epic should close (only child).
	if len(changes) != 1 {
		t.Fatalf("len(changes) = %d, want 1", len(changes))
	}

	c, _ := store.Get("t-c1")
	if c.Status != StatusClosed {
		t.Errorf("child status = %q, want closed", c.Status)
	}
}

func TestSetStatus_InvalidStatus(t *testing.T) {
	store := NewFileStore(t.TempDir())
	tk := wfTicket("t-1", StatusOpen, "")
	if err := store.Create(tk); err != nil {
		t.Fatal(err)
	}

	_, err := SetStatus(store, "t-1", "invalid")
	if err == nil {
		t.Error("SetStatus with invalid status should fail")
	}
}

func TestPropagateStatus_ParentAlreadyClosed(t *testing.T) {
	store := NewFileStore(t.TempDir())
	epic := wfTicket("t-epic", StatusClosed, "")
	epic.Type = TypeEpic
	child := wfTicket("t-c1", StatusClosed, "t-epic")

	for _, tk := range []*Ticket{epic, child} {
		if err := store.Create(tk); err != nil {
			t.Fatal(err)
		}
	}

	changes, err := PropagateStatus(store, "t-c1")
	if err != nil {
		t.Fatalf("PropagateStatus: %v", err)
	}
	// Parent already closed, no change needed.
	if len(changes) != 0 {
		t.Errorf("expected no changes when parent already closed, got %d", len(changes))
	}
}
