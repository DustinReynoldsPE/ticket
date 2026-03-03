---
id: add-conversation-session-c401
stage: triage
status: closed
deps: []
links: []
created: 2026-03-03T15:04:18Z
type: feature
priority: 0
---
# Add conversation/session tracking to tickets



## Notes

**2026-03-03T15:04:30Z**

The `conversations` field exists in ticket frontmatter but is underutilized. Add support for linking tickets to Claude Code session IDs so a supervisor can trace which agent session produced which work. This likely means: a `tk link-session <id> <session-id>` command (or automatic linking during claim/edit), and surfacing the conversation list in `tk show` output.

**2026-03-03T22:04:24Z**

<!-- checkpoint: brainstorm -->
## Brainstorm
**Decision:** (auto) Add tk link-session CLI + ticket_link_session MCP tool. Verify tk show displays conversations. No auto-linking — keep explicit.

**2026-03-03T22:04:45Z**

<!-- checkpoint: planning -->
## Plan
1. cmd/link_session.go — CLI command
2. internal/mcp/mcp.go — ticket_link_session MCP tool
3. cmd/root.go, README.md, CHANGELOG.md updates
4. MCP tests

**2026-03-03T22:06:07Z**

<!-- checkpoint: finalized -->
