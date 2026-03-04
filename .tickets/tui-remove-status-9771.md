---
id: tui-remove-status-9771
stage: triage
status: open
deps: [core-remove-status-17f0]
links: []
created: 2026-03-04T05:02:38Z
type: task
priority: 2
parent: remove-status-stage-7439
version: 2
---
# TUI: Remove status references

Remove all status references from the TUI, use stage exclusively.

## Design

Files: internal/tui/tui.go, internal/tui/detail.go, internal/tui/pipeline.go, internal/tui/form.go, internal/tui/dashboard.go

tui.go:
- New ticket creation: remove Status assignment, keep Stage: StageTriage
- Remove any Status display logic

detail.go:
- Remove Status field display (keep Stage display)

pipeline.go:
- Remove legacy status→stage mapping for display

dashboard.go:
- Remove status-based filters, use stage exclusively (done/triage checks already use stage)

form.go:
- Remove status from edit form if present (stage field should remain)

## Acceptance Criteria

1. No .Status references in internal/tui/
2. New tickets only set stage
3. All views display stage, not status
