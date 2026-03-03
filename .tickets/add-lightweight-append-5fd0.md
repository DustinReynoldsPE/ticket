---
id: add-lightweight-append-5fd0
stage: triage
status: closed
deps: []
links: []
created: 2026-03-03T15:04:18Z
type: feature
priority: 0
---
# Add lightweight append-only event log



## Notes

**2026-03-03T15:04:27Z**

Add an append-only log file at `.tickets/.log` that records one line per state transition: `[timestamp] [ticket-id] [old-stage] → [new-stage] [agent]`. Pure append means no merge conflicts. Gives supervisors a stream of events without needing to parse every ticket file. Should be written by the store layer on any status/stage change.

**2026-03-03T22:00:40Z**

<!-- checkpoint: brainstorm -->
## Brainstorm
**Decision:** (auto) Log create/delete/stage/status/claim events via store-level AppendLog. Agent identity as field on FileStore. POSIX O_APPEND, no locking needed.

**2026-03-03T22:00:59Z**

<!-- checkpoint: planning -->
## Plan
1. pkg/ticket/store.go — appendLog() method writing to .log in tickets dir. Called from Create(), Update() (diff stage/status), Delete().
2. Log format: timestamp ticket-id event [details]. No agent field (assignee already on ticket).
3. pkg/ticket/store_test.go — verify log entries
4. CHANGELOG.md update

**2026-03-03T22:02:55Z**

<!-- checkpoint: finalized -->
