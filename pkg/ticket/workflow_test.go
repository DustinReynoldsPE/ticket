package ticket

import (
	"strings"
	"testing"
	"time"
)

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

	updated, _ := store.Get("t-adv")
	if updated.Stage != StageImplement {
		t.Errorf("persisted stage = %s, want implement", updated.Stage)
	}
}

func TestAdvance_GateFails(t *testing.T) {
	store := NewFileStore(t.TempDir())
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

func TestAdvance_ForceCannotReachDone(t *testing.T) {
	store := NewFileStore(t.TempDir())
	tk := stageTicket("t-fdone", StageVerify, TypeTask)
	if err := store.Create(tk); err != nil {
		t.Fatal(err)
	}

	_, err := Advance(store, "t-fdone", AdvanceOptions{Force: true})
	if err == nil {
		t.Fatal("Force advance to done should fail")
	}
	if !strings.Contains(err.Error(), "cannot force-advance to done") && !strings.Contains(err.Error(), "use 'tk skip'") {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify ticket did not advance.
	check, _ := store.Get("t-fdone")
	if check.Stage != StageVerify {
		t.Errorf("stage should still be verify, got %s", check.Stage)
	}
}

func TestAdvance_ForceCannotSkipToDone(t *testing.T) {
	store := NewFileStore(t.TempDir())
	tk := stageTicket("t-fskip", StageTriage, TypeChore)
	if err := store.Create(tk); err != nil {
		t.Fatal(err)
	}

	_, err := Advance(store, "t-fskip", AdvanceOptions{
		Force:  true,
		SkipTo: StageDone,
		Reason: "skip it all",
	})
	if err == nil {
		t.Fatal("Force skip to done should fail")
	}
	if !strings.Contains(err.Error(), "cannot force-advance to done") && !strings.Contains(err.Error(), "use 'tk skip'") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAdvance_BlockedByDeps(t *testing.T) {
	store := NewFileStore(t.TempDir())
	dep := stageTicket("t-dep", StageTriage, TypeTask)
	tk := stageTicket("t-blocked", StageTriage, TypeChore)
	tk.Deps = []string{"t-dep"}

	for _, x := range []*Ticket{dep, tk} {
		if err := store.Create(x); err != nil {
			t.Fatal(err)
		}
	}

	_, err := Advance(store, "t-blocked", AdvanceOptions{})
	if err == nil {
		t.Fatal("Advance on blocked ticket should fail")
	}
	if !strings.Contains(err.Error(), "blocked") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAdvance_BlockedNotBypassedByForce(t *testing.T) {
	store := NewFileStore(t.TempDir())
	dep := stageTicket("t-dep2", StageImplement, TypeTask)
	tk := stageTicket("t-blk2", StageTriage, TypeChore)
	tk.Deps = []string{"t-dep2"}

	for _, x := range []*Ticket{dep, tk} {
		if err := store.Create(x); err != nil {
			t.Fatal(err)
		}
	}

	_, err := Advance(store, "t-blk2", AdvanceOptions{Force: true})
	if err == nil {
		t.Fatal("Force should not bypass dep blocking")
	}
	if !strings.Contains(err.Error(), "blocked") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAdvance_UnblockedWhenDepsDone(t *testing.T) {
	store := NewFileStore(t.TempDir())
	dep := stageTicket("t-dep3", StageDone, TypeTask)
	tk := stageTicket("t-unblk", StageTriage, TypeChore)
	tk.Deps = []string{"t-dep3"}

	for _, x := range []*Ticket{dep, tk} {
		if err := store.Create(x); err != nil {
			t.Fatal(err)
		}
	}

	result, err := Advance(store, "t-unblk", AdvanceOptions{})
	if err != nil {
		t.Fatalf("Advance on unblocked ticket: %v", err)
	}
	if result.To != StageImplement {
		t.Errorf("advance to = %s, want implement", result.To)
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

func TestSkip_ToDone(t *testing.T) {
	store := NewFileStore(t.TempDir())
	tk := stageTicket("t-skip-done", StageVerify, TypeTask)
	if err := store.Create(tk); err != nil {
		t.Fatal(err)
	}

	result, err := Skip(store, "t-skip-done", StageDone, "verified manually")
	if err != nil {
		t.Fatalf("Skip to done: %v", err)
	}
	if result.To != StageDone {
		t.Errorf("skip to = %s, want done", result.To)
	}

	updated, _ := store.Get("t-skip-done")
	if updated.Stage != StageDone {
		t.Errorf("persisted stage = %s, want done", updated.Stage)
	}
}

func TestSkip_ToDoneFromEarly(t *testing.T) {
	store := NewFileStore(t.TempDir())
	tk := stageTicket("t-skip-early", StageTriage, TypeTask)
	if err := store.Create(tk); err != nil {
		t.Fatal(err)
	}

	result, err := Skip(store, "t-skip-early", StageDone, "not needed")
	if err != nil {
		t.Fatalf("Skip to done from triage: %v", err)
	}
	if result.To != StageDone {
		t.Errorf("skip to = %s, want done", result.To)
	}
	// Should have skipped implement, test, verify
	if len(result.Skipped) != 3 {
		t.Errorf("skipped %d stages, want 3", len(result.Skipped))
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
	tk := stageTicket("t-rev2", StageDesign, TypeFeature)
	if err := store.Create(tk); err != nil {
		t.Fatal(err)
	}

	err := SetReview(store, "t-rev2", "human:steve", ReviewApproved, "looks good")
	if err != nil {
		t.Fatalf("SetReview: %v", err)
	}

	updated, _ := store.Get("t-rev2")
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

func TestPropagateStage_NoParent(t *testing.T) {
	store := NewFileStore(t.TempDir())
	tk := stageTicket("t-1", StageDone, TypeTask)
	if err := store.Create(tk); err != nil {
		t.Fatal(err)
	}

	changes, err := PropagateStage(store, "t-1")
	if err != nil {
		t.Fatalf("PropagateStage: %v", err)
	}
	if len(changes) != 0 {
		t.Errorf("expected no changes for orphan ticket, got %d", len(changes))
	}
}

func TestPropagateStage_Recursive(t *testing.T) {
	store := NewFileStore(t.TempDir())
	grandparent := stageTicket("t-gp", StageDesign, TypeEpic)
	parent := stageTicket("t-p", StageDesign, TypeEpic)
	parent.Parent = "t-gp"
	child := stageTicket("t-c", StageDone, TypeTask)
	child.Parent = "t-p"

	for _, tk := range []*Ticket{grandparent, parent, child} {
		if err := store.Create(tk); err != nil {
			t.Fatal(err)
		}
	}

	changes, err := PropagateStage(store, "t-c")
	if err != nil {
		t.Fatalf("PropagateStage: %v", err)
	}
	if len(changes) != 2 {
		t.Fatalf("len(changes) = %d, want 2", len(changes))
	}

	gp, _ := store.Get("t-gp")
	if gp.Stage != StageDone {
		t.Errorf("grandparent stage = %s, want done", gp.Stage)
	}
}

func TestPropagateStage_ParentAlreadyDone(t *testing.T) {
	store := NewFileStore(t.TempDir())
	epic := stageTicket("t-epic", StageDone, TypeEpic)
	child := stageTicket("t-c1", StageDone, TypeTask)
	child.Parent = "t-epic"

	for _, tk := range []*Ticket{epic, child} {
		if err := store.Create(tk); err != nil {
			t.Fatal(err)
		}
	}

	changes, err := PropagateStage(store, "t-c1")
	if err != nil {
		t.Fatalf("PropagateStage: %v", err)
	}
	if len(changes) != 0 {
		t.Errorf("expected no changes when parent already done, got %d", len(changes))
	}
}

func TestPropagateStage_MixedNoChange(t *testing.T) {
	store := NewFileStore(t.TempDir())
	epic := stageTicket("t-epic", StageDesign, TypeEpic)
	child1 := stageTicket("t-c1", StageDone, TypeTask)
	child1.Parent = "t-epic"
	child2 := stageTicket("t-c2", StageTriage, TypeTask)
	child2.Parent = "t-epic"

	for _, tk := range []*Ticket{epic, child1, child2} {
		if err := store.Create(tk); err != nil {
			t.Fatal(err)
		}
	}

	changes, err := PropagateStage(store, "t-c1")
	if err != nil {
		t.Fatalf("PropagateStage: %v", err)
	}
	if len(changes) != 0 {
		t.Errorf("expected no changes for mixed stages, got %d", len(changes))
	}
}
