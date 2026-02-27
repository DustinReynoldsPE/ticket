---
id: gate-checks-require-0621
stage: triage
status: open
deps: []
links: []
created: 2026-02-27T07:42:26Z
type: bug
priority: 1
assignee: Steve Macbeth
tags: [mcp, gates]
---
# Gate checks require body sections but ticket_edit writes to frontmatter


Gates at spec->design, design->implement, and test->verify check for markdown body sections (## Acceptance Criteria, ## Design, ## Test Results) but ticket_edit writes acceptance and design fields to YAML frontmatter only. This forces manual editing of .tickets/ files to pass gates, breaking the MCP-only workflow. Either gates should check frontmatter fields, or ticket_edit should write body sections, or both.
