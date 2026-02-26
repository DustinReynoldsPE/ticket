package ticket

import (
	"testing"
	"time"
)

func TestNeedsMigration(t *testing.T) {
	// Legacy ticket with status, no stage.
	legacy := &Ticket{Status: StatusOpen}
	if !NeedsMigration(legacy) {
		t.Error("legacy ticket should need migration")
	}

	// Already migrated.
	migrated := &Ticket{Stage: StageTriage, Status: StatusOpen}
	if NeedsMigration(migrated) {
		t.Error("ticket with stage should not need migration")
	}

	// Stage-only ticket.
	stageOnly := &Ticket{Stage: StageImplement}
	if NeedsMigration(stageOnly) {
		t.Error("stage-only ticket should not need migration")
	}
}

func TestMigrateTicket_AllMappings(t *testing.T) {
	tests := []struct {
		status Status
		want   Stage
	}{
		{StatusOpen, StageTriage},
		{StatusInProgress, StageImplement},
		{StatusNeedsTesting, StageTest},
		{StatusClosed, StageDone},
	}

	for _, tt := range tests {
		tk := &Ticket{
			ID:       "t-test",
			Status:   tt.status,
			Type:     TypeTask,
			Priority: 2,
			Deps:     []string{},
			Links:    []string{},
			Created:  time.Now(),
		}
		changed := MigrateTicket(tk)
		if !changed {
			t.Errorf("MigrateTicket(%s) returned false", tt.status)
		}
		if tk.Stage != tt.want {
			t.Errorf("MigrateTicket(%s): stage = %s, want %s", tt.status, tk.Stage, tt.want)
		}
		// Status should be preserved for backward compat.
		if tk.Status != tt.status {
			t.Errorf("MigrateTicket(%s): status was cleared", tt.status)
		}
	}
}

func TestMigrateTicket_Idempotent(t *testing.T) {
	tk := &Ticket{
		ID:     "t-test",
		Status: StatusOpen,
		Stage:  StageTriage,
		Type:   TypeTask,
	}
	changed := MigrateTicket(tk)
	if changed {
		t.Error("MigrateTicket on already-migrated ticket should return false")
	}
}

func TestMigrateAll(t *testing.T) {
	store := NewFileStore(t.TempDir())

	// Create a mix of legacy and stage-based tickets.
	legacy1 := &Ticket{
		ID: "t-1", Status: StatusOpen, Type: TypeTask, Priority: 2,
		Deps: []string{}, Links: []string{}, Created: time.Now(), Title: "Legacy 1", Body: "\n",
	}
	legacy2 := &Ticket{
		ID: "t-2", Status: StatusClosed, Type: TypeBug, Priority: 1,
		Deps: []string{}, Links: []string{}, Created: time.Now(), Title: "Legacy 2", Body: "\n",
	}
	modern := &Ticket{
		ID: "t-3", Stage: StageDesign, Type: TypeFeature, Priority: 0,
		Deps: []string{}, Links: []string{}, Created: time.Now(), Title: "Modern", Body: "\n",
	}

	for _, tk := range []*Ticket{legacy1, legacy2, modern} {
		if err := store.Create(tk); err != nil {
			t.Fatal(err)
		}
	}

	count, err := MigrateAll(store)
	if err != nil {
		t.Fatalf("MigrateAll: %v", err)
	}
	if count != 2 {
		t.Errorf("MigrateAll migrated %d tickets, want 2", count)
	}

	// Verify.
	t1, _ := store.Get("t-1")
	if t1.Stage != StageTriage {
		t.Errorf("t-1 stage = %s, want triage", t1.Stage)
	}
	t2, _ := store.Get("t-2")
	if t2.Stage != StageDone {
		t.Errorf("t-2 stage = %s, want done", t2.Stage)
	}
	t3, _ := store.Get("t-3")
	if t3.Stage != StageDesign {
		t.Errorf("t-3 stage = %s, want design (unchanged)", t3.Stage)
	}
}
