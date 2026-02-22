---
id: tic-303f
status: open
deps: []
links: []
created: 2026-02-22T22:02:20Z
type: task
priority: 4
assignee: Steve Macbeth
parent: tic-18e9
tags: [go-parity]
---
# Add migrate-beads command


migrate-beads command missing from Go version. Imports tickets from .beads/issues.jsonl format.

## Design

Files: cmd/migrate.go
Read .beads/issues.jsonl, map fields to ticket format:
- id, status, title, description, design, acceptance_criteria, notes
- created_at -> created, issue_type -> type, priority, assignee, external_ref
- dependencies with type blocks -> deps, related -> links, parent-child -> parent
Use encoding/json to parse, create tickets via FileStore.

## Acceptance Criteria

tk migrate-beads imports all tickets from .beads/issues.jsonl with correct field mapping.
