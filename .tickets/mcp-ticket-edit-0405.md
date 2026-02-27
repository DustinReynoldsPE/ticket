---
id: mcp-ticket-edit-0405
stage: triage
status: open
deps: []
links: []
created: 2026-02-27T08:00:51Z
type: bug
priority: 1
assignee: Steve Macbeth
tags: [mcp]
---
# MCP ticket_edit silently drops body fields (acceptance, design, description)


When calling ticket_edit via MCP with acceptance, design, or description fields, the edit reports success but the values are not persisted to the ticket body. The CLI edit command writes these correctly. The MCP handler likely updates frontmatter only and doesn't rebuild the markdown body sections.
