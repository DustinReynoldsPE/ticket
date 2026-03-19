---
id: remove-status-stage-7439
stage: done
status: open
deps: []
links: []
created: 2026-03-04T05:01:51Z
type: epic
priority: 2
version: 6
---
# Remove Status, Stage-only state model

Remove the Status field entirely from tk. Stage becomes the sole state model. Breaking change for v3.0.0.

## Design

Auto-migrate on read: if ticket has status but no stage, map silently (open→triage, in_progress→implement, needs_testing→test, closed→done). Never write status back.

Deletions: Status type/constants/ValidateStatus(), status/start/close/reopen CLI commands, --status flag, status MCP params, PropagateStatus()/SetStatus(), migrate command, statusOrder().

Additions: --stage flag on ls, Stage field in ListOptions, stageOrder() function, legacy status map in format.go (4 entries, unexported).

Changes: Validate() requires stage, Filter() uses Stage, ls excludes stage:done, MCP uses stage not status, stats/show/TUI stage-only.

~15 files modified, ~4 files deleted, net code reduction.

## Acceptance Criteria

1. Status type and all references removed from codebase
2. Tickets with legacy status: field auto-migrate to stage on read
3. All CLI commands use stage exclusively
4. MCP tools use stage exclusively
5. All tests pass
6. README and CHANGELOG updated
