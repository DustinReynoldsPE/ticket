---
id: core-remove-status-17f0
stage: done
status: open
deps: []
links: []
created: 2026-03-04T05:02:01Z
type: task
priority: 2
parent: remove-status-stage-7439
version: 2
---
# Core: Remove Status type, add auto-migrate on read

Remove Status type and add legacy auto-migration in the parser. This is the foundation all other tasks depend on.

## Design

Files: pkg/ticket/ticket.go, pkg/ticket/format.go

ticket.go:
- Delete type Status string, StatusOpen/StatusInProgress/StatusNeedsTesting/StatusClosed constants
- Delete validStatuses map and ValidateStatus() function
- Remove Status field from Ticket struct
- Update Validate(): require Stage (not 'status or stage')

format.go:
- Add unexported legacyStatusToStage map: {open→triage, in_progress→implement, needs_testing→test, closed→done}
- In parse path after YAML unmarshal: if Stage=='' && status field present in raw YAML, map it. Always clear status from struct.
- Stop writing status: field in serialize path

## Acceptance Criteria

1. Status type does not exist in ticket.go
2. Ticket struct has no Status field
3. Validate() requires stage
4. format.go auto-migrates status→stage on read
5. format.go never writes status:
6. Compiles (other packages will break — expected)
