---
id: core-remove-status-3402
stage: triage
status: open
deps: [core-remove-status-17f0]
links: []
created: 2026-03-04T05:02:15Z
type: task
priority: 2
parent: remove-status-stage-7439
version: 2
---
# Core: Remove status from workflow

Remove all status-related workflow functions and migration code.

## Design

Files: pkg/ticket/workflow.go, pkg/ticket/migrate.go, pkg/ticket/store.go

workflow.go:
- Delete PropagateStatus() function
- Delete SetStatus() function
- Delete to_done_compat() — replace call sites with direct t.Stage = StageDone
- Remove all t.Status assignments from PropagateStage()

migrate.go:
- Delete StatusToStage map, NeedsMigration(), MigrateTicket(), MigrateAll()
- Delete the file entirely if nothing remains

store.go:
- Remove status change logging (lines tracking old/new status)
- Keep stage change logging only

## Acceptance Criteria

1. No PropagateStatus(), SetStatus(), or to_done_compat() functions
2. migrate.go deleted or empty
3. store.go only logs stage transitions
4. PropagateStage() has no status references
