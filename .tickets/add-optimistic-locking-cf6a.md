---
id: add-optimistic-locking-cf6a
stage: triage
status: closed
deps: []
links: []
created: 2026-03-03T15:04:17Z
type: feature
priority: 0
---
# Add optimistic locking via version counter



## Notes

**2026-03-03T15:04:25Z**

Add a `version` counter (or `modified` timestamp) to ticket frontmatter. On every `Update()` call, compare the in-memory version against the on-disk version — if they differ, abort with a conflict error instead of silently overwriting. This prevents data loss when multiple agents or humans edit the same ticket concurrently. Increment the counter on each successful write.

**2026-03-03T21:55:36Z**

<!-- checkpoint: brainstorm -->
## Brainstorm
**Decision:** (auto) Integer version counter in frontmatter, enforced in FileStore.Update(). ErrConflict error type. Legacy tickets default to version 0. No migration needed.

**2026-03-03T21:55:56Z**

<!-- checkpoint: planning -->
## Plan
1. pkg/ticket/ticket.go — Add Version int field
2. pkg/ticket/store.go — ErrConflict type, version check in Update(), set version=1 in Create()
3. internal/mcp/mcp.go — Add version to ticketJSON
4. MCP tests for conflict detection
5. CHANGELOG.md update

**2026-03-03T21:59:36Z**

<!-- checkpoint: finalized -->
