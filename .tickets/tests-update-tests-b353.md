---
id: tests-update-tests-b353
stage: triage
status: open
deps: [core-replace-status-83c2, core-remove-status-3402, cli-remove-status-cc0e, mcp-replace-status-5712, tui-remove-status-9771, cli-update-show-98ca]
links: []
created: 2026-03-04T05:02:54Z
type: task
priority: 2
parent: remove-status-stage-7439
version: 7
---
# Tests: Update all tests for stage-only model

Update all test files to remove status references and add auto-migrate coverage.

## Design

Files: pkg/ticket/*_test.go, internal/mcp/mcp_test.go

- Remove all Status field assignments in test ticket construction
- Remove status-based test assertions
- Remove migrate_test.go (or repurpose for auto-migrate-on-read tests)
- Add test: parse ticket with status: open, no stage → gets stage: triage
- Add test: parse ticket with status: closed, no stage → gets stage: done
- Add test: parse ticket with both status and stage → stage wins, status ignored
- Add test: serialize ticket → no status: field in output
- Update MCP test assertions to check stage instead of status
- Update filter/sort tests for stage-based logic
- Run go test ./... and verify all pass

## Acceptance Criteria

1. go test ./... passes with 0 failures
2. No Status references in test files (except auto-migrate test inputs)
3. Auto-migrate-on-read has explicit test coverage
