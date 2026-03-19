---
id: docs-update-readme-7296
stage: done
status: open
deps: [tests-update-tests-b353]
links: []
created: 2026-03-04T05:03:03Z
type: task
priority: 2
parent: remove-status-stage-7439
skipped: [implement, test, verify]
version: 3
---
# Docs: Update README and CHANGELOG for v3.0.0

Update documentation to reflect status removal and v3.0.0 breaking change.

## Design

Files: README.md, CHANGELOG.md

README.md:
- Remove all status references (open, in_progress, needs_testing, closed)
- Remove --status from filter flags table
- Add --stage to filter flags table
- Remove status/start/close/reopen from command list
- Remove migrate from command list
- Update examples that reference status

CHANGELOG.md:
- Add ## [3.0.0] - <date> section
- Breaking: Removed Status field, status/start/close/reopen/migrate commands, --status flag
- Added: --stage filter flag on ls
- Changed: Auto-migrate legacy status→stage on read
- Changed: MCP tools use stage instead of status

## Acceptance Criteria

1. No status references in README (except historical context if any)
2. CHANGELOG has v3.0.0 section documenting all breaking changes
3. --stage flag documented
