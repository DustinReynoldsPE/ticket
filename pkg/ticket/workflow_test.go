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

// --- Stage pipeline tests ---

func stageTicket(id string, stage Stage, ticketType TicketType) *Ticket {
	return &Ticket{
		ID:       id,
		Stage:    stage,
		Type:     ticketType,
		Priority: 2,
		Deps:     []string{},
		Links:    []string{},
		Created:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Title:    "Ticket " + id,
		Body:     "\nDescription.\n",
	}
}

func TestAdvance_NextStage(t *testing.T) {
	store := NewFileStore(t.TempDir())
	tk := stageTicket("t-adv", StageTriage, TypeChore)
	if err := store.Create(tk); err != nil {
		t.Fatal(err)
	}

	result, err := Advance(store, "t-adv", AdvanceOptions{})
	if err != nil {
		t.Fatalf("Advance: %v", err)
	}
	if result.From != StageTriage || result.To != StageImplement {
		t.Errorf("Advance = %s→%s, want triage→implement", result.From, result.To)
	}

	// Verify persisted.
	updated, _ := store.Get("t-adv")
	if updated.Stage != StageImplement {
		t.Errorf("persisted stage = %s, want implement", updated.Stage)
	}
}

func TestAdvance_GateFails(t *testing.T) {
	store := NewFileStore(t.TempDir())
	// Feature at spec, no AC, no review approval → should fail spec→design.
	tk := stageTicket("t-gate", StageSpec, TypeFeature)
	if err := store.Create(tk); err != nil {
		t.Fatal(err)
	}

	result, err := Advance(store, "t-gate", AdvanceOptions{})
	if err == nil {
		t.Fatal("Advance should fail when gates fail")
	}
	if len(result.GateErrors) == 0 {
		t.Error("GateErrors should be populated")
	}

	// Verify ticket did not advance.
	check, _ := store.Get("t-gate")
	if check.Stage != StageSpec {
		t.Errorf("stage should still be spec, got %s", check.Stage)
	}
}

func TestAdvance_Force(t *testing.T) {
	store := NewFileStore(t.TempDir())
	tk := stageTicket("t-force", StageSpec, TypeFeature)
	if err := store.Create(tk); err != nil {
		t.Fatal(err)
	}

	result, err := Advance(store, "t-force", AdvanceOptions{Force: true})
	if err != nil {
		t.Fatalf("Advance with Force: %v", err)
	}
	if result.To != StageDesign {
		t.Errorf("forced advance to = %s, want design", result.To)
	}
}

func TestAdvance_AlreadyAtFinal(t *testing.T) {
	store := NewFileStore(t.TempDir())
	tk := stageTicket("t-done", StageDone, TypeChore)
	if err := store.Create(tk); err != nil {
		t.Fatal(err)
	}

	_, err := Advance(store, "t-done", AdvanceOptions{})
	if err == nil {
		t.Error("Advance at done should fail")
	}
}

func TestAdvance_ResetsReview(t *testing.T) {
	store := NewFileStore(t.TempDir())
	tk := stageTicket("t-rev", StageTriage, TypeChore)
	tk.Review = ReviewApproved
	if err := store.Create(tk); err != nil {
		t.Fatal(err)
	}

	Advance(store, "t-rev", AdvanceOptions{})
	updated, _ := store.Get("t-rev")
	if updated.Review != ReviewNone {
		t.Errorf("review should reset to empty after advance, got %s", updated.Review)
	}
}

func TestSkip(t *testing.T) {
	store := NewFileStore(t.TempDir())
	tk := stageTicket("t-skip", StageTriage, TypeFeature)
	if err := store.Create(tk); err != nil {
		t.Fatal(err)
	}

	result, err := Skip(store, "t-skip", StageImplement, "trivial feature")
	if err != nil {
		t.Fatalf("Skip: %v", err)
	}
	if result.To != StageImplement {
		t.Errorf("skip to = %s, want implement", result.To)
	}
	if len(result.Skipped) != 2 { // skipped spec and design
		t.Errorf("skipped %d stages, want 2", len(result.Skipped))
	}

	updated, _ := store.Get("t-skip")
	if len(updated.Skipped) != 2 {
		t.Errorf("persisted skipped = %d, want 2", len(updated.Skipped))
	}
}

func TestSkip_RequiresReason(t *testing.T) {
	store := NewFileStore(t.TempDir())
	tk := stageTicket("t-skip2", StageTriage, TypeFeature)
	if err := store.Create(tk); err != nil {
		t.Fatal(err)
	}

	_, err := Skip(store, "t-skip2", StageImplement, "")
	if err == nil {
		t.Error("Skip without reason should fail")
	}
}

func TestSkip_CannotGoBackward(t *testing.T) {
	store := NewFileStore(t.TempDir())
	tk := stageTicket("t-back", StageImplement, TypeFeature)
	if err := store.Create(tk); err != nil {
		t.Fatal(err)
	}

	_, err := Skip(store, "t-back", StageTriage, "oops")
	if err == nil {
		t.Error("Skip backward should fail")
	}
}

func TestSetReview(t *testing.T) {
	store := NewFileStore(t.TempDir())
	tk := stageTicket("t-rev", StageDesign, TypeFeature)
	if err := store.Create(tk); err != nil {
		t.Fatal(err)
	}

	err := SetReview(store, "t-rev", "human:steve", ReviewApproved, "looks good")
	if err != nil {
		t.Fatalf("SetReview: %v", err)
	}

	updated, _ := store.Get("t-rev")
	if updated.Review != ReviewApproved {
		t.Errorf("review = %s, want approved", updated.Review)
	}
	if len(updated.Reviews) != 1 {
		t.Fatalf("len(Reviews) = %d, want 1", len(updated.Reviews))
	}
	if updated.Reviews[0].Reviewer != "human:steve" {
		t.Errorf("reviewer = %s, want human:steve", updated.Reviews[0].Reviewer)
	}
	if updated.Reviews[0].Comment != "looks good" {
		t.Errorf("comment = %s, want 'looks good'", updated.Reviews[0].Comment)
	}
}

func TestPropagateStage_AllDone(t *testing.T) {
	store := NewFileStore(t.TempDir())
	epic := stageTicket("t-epic", StageDesign, TypeEpic)
	child1 := stageTicket("t-c1", StageDone, TypeTask)
	child1.Parent = "t-epic"
	child2 := stageTicket("t-c2", StageDone, TypeTask)
	child2.Parent = "t-epic"

	for _, tk := range []*Ticket{epic, child1, child2} {
		if err := store.Create(tk); err != nil {
			t.Fatal(err)
		}
	}

	changes, err := PropagateStage(store, "t-c2")
	if err != nil {
		t.Fatalf("PropagateStage: %v", err)
	}
	if len(changes) != 1 {
		t.Fatalf("len(changes) = %d, want 1", len(changes))
	}
	if changes[0].NewStage != StageDone {
		t.Errorf("new stage = %s, want done", changes[0].NewStage)
	}
}
