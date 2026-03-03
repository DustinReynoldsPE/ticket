---
id: add-claim-command-1be4
stage: triage
status: in_progress
deps: []
links: []
created: 2026-03-03T15:04:17Z
type: feature
priority: 0
---
# Add claim command with assignee enforcement



## Notes

**2026-03-03T15:04:23Z**

Add a `tk claim <id>` command that sets the assignee field but fails if the ticket is already assigned to someone else. The assignee field exists in frontmatter today but is purely informational — this adds enforcement logic to prevent two agents from picking up the same ticket. Should also add a `--force` flag for supervisor override.

**2026-03-03T21:48:52Z**

<!-- checkpoint: brainstorm -->
## Brainstorm
**Decision:** (auto) Default assignee from git user.name, --assignee flag overrides. Self-claim is no-op. No status coupling. MCP tool included.

**2026-03-03T21:49:27Z**

<!-- checkpoint: planning -->
## Plan
1. pkg/ticket/workflow.go — Claim() function with enforcement logic
2. cmd/claim.go — CLI command with --assignee and --force flags
3. internal/mcp/mcp.go — ticket_claim MCP tool
4. cmd/root.go — help text update
5. CHANGELOG.md, README.md updates

**2026-03-03T21:53:02Z**

<!-- checkpoint: testing -->
## Test Results
- All go tests pass (go test ./...)
- CLI smoke tests pass: claim, conflict rejection, force override, self-claim no-op
- MCP tests: 4 new tests covering claim, conflict, force, self-claim
