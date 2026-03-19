package ticket

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMigrateAll_LegacyStatusToStage(t *testing.T) {
	dir := t.TempDir()
	store := NewFileStore(dir)

	// Write a legacy ticket file with status but no stage.
	legacy := `---
id: t-1
status: open
deps: []
links: []
created: 2026-01-01T00:00:00Z
type: task
priority: 2
---
# Legacy ticket

Description.
`
	if err := os.WriteFile(filepath.Join(dir, "t-1.md"), []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}

	// Write a modern ticket with stage.
	modern := &Ticket{
		ID: "t-2", Stage: StageDesign, Type: TypeFeature, Priority: 0,
		Deps: []string{}, Links: []string{}, Created: time.Now(), Title: "Modern", Body: "\n",
	}
	if err := store.Create(modern); err != nil {
		t.Fatal(err)
	}

	count, err := MigrateAll(store)
	if err != nil {
		t.Fatalf("MigrateAll: %v", err)
	}
	if count != 2 {
		t.Errorf("MigrateAll migrated %d tickets, want 2", count)
	}

	// Verify legacy ticket now has stage and no status in file.
	t1, err := store.Get("t-1")
	if err != nil {
		t.Fatalf("Get t-1: %v", err)
	}
	if t1.Stage != StageTriage {
		t.Errorf("t-1 stage = %s, want triage", t1.Stage)
	}

	// Verify the file no longer contains "status:".
	data, _ := os.ReadFile(filepath.Join(dir, "t-1.md"))
	if strings.Contains(string(data), "status:") {
		t.Error("migrated file should not contain status field")
	}

	// Modern ticket unchanged.
	t2, _ := store.Get("t-2")
	if t2.Stage != StageDesign {
		t.Errorf("t-2 stage = %s, want design (unchanged)", t2.Stage)
	}
}

func TestParse_AllLegacyStatusMappings(t *testing.T) {
	tests := []struct {
		status string
		want   Stage
	}{
		{"open", StageTriage},
		{"in_progress", StageImplement},
		{"needs_testing", StageTest},
		{"closed", StageDone},
	}

	for _, tt := range tests {
		input := "---\nid: t-test\nstatus: " + tt.status + "\ndeps: []\nlinks: []\ncreated: 2026-01-01T00:00:00Z\ntype: task\npriority: 2\n---\n# Test\n"
		tk, err := Parse(strings.NewReader(input))
		if err != nil {
			t.Errorf("Parse(status=%s): %v", tt.status, err)
			continue
		}
		if tk.Stage != tt.want {
			t.Errorf("Parse(status=%s): stage = %s, want %s", tt.status, tk.Stage, tt.want)
		}
	}
}
