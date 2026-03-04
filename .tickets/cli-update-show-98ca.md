---
id: cli-update-show-98ca
stage: done
status: open
deps: [core-remove-status-17f0]
links: []
created: 2026-03-04T05:02:45Z
type: task
priority: 2
parent: remove-status-stage-7439
version: 3
---
# CLI: Update show, stats, query, and remaining commands

Remove status references from all remaining CLI commands.

## Design

Files: cmd/show.go, cmd/stats.go, cmd/closed.go, cmd/timeline.go, cmd/dep.go, cmd/log.go, cmd/pipeline.go, cmd/query.go, cmd/create.go

show.go: Remove status display lines, keep stage display. Remove status-based blocker checks (use stage only).

stats.go: Replace status histogram with stage histogram. Remove status-based filtering.

closed.go: Filter by stage==done instead of status==closed.

timeline.go: Filter by stage==done instead of status==closed.

dep.go: Remove status references in dependency display.

log.go: Remove status display, show stage only.

pipeline.go: Remove legacy status→stage mapping.

query.go: Remove status from JSON output if present.

create.go: Remove Status: StatusOpen assignment, keep Stage: StageTriage only.

## Acceptance Criteria

1. No .Status references in any cmd/ file
2. stats shows stage histogram
3. closed/timeline filter by StageDone
4. show displays stage only
