---
id: cli-remove-status-cc0e
stage: done
status: open
deps: [core-remove-status-17f0]
links: []
created: 2026-03-04T05:02:23Z
type: task
priority: 2
parent: remove-status-stage-7439
version: 3
---
# CLI: Remove status commands and flags

Delete status CLI commands, replace --status flag with --stage on ls, remove from edit.

## Design

Files: cmd/status.go (delete), cmd/ls.go, cmd/edit.go, cmd/migrate.go (delete), cmd/root.go

status.go: Delete entirely (status, start, close, reopen commands)

migrate.go: Delete entirely (migrate command)

ls.go:
- Remove --status flag
- Add --stage flag (string, filter by stage)
- Update filtering: if --stage set, opts.Stage = ticket.Stage(val); else exclude StageDone
- Update call to renamed sort function (SortByStagePriorityID)

edit.go:
- Remove --status/-s flag and its processing block
- Keep --stage flag (already exists)

root.go:
- Remove status/start/close/reopen/migrate from command registration
- Update help text to remove status references

## Acceptance Criteria

1. status.go and migrate.go deleted
2. No --status flag anywhere
3. ls has --stage flag that filters correctly
4. edit only has --stage, no --status
5. Help text has no status references
