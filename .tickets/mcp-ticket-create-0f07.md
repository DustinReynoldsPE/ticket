---
id: mcp-ticket-create-0f07
stage: triage
status: open
deps: []
links: []
created: 2026-02-27T07:09:25Z
type: bug
priority: 1
assignee: Steve Macbeth
tags: [mcp]
---
# MCP ticket_create fails with ticket ID is required


The ticket_create MCP tool fails with 'create: ticket ID is required' when called with a title. The title parameter is not being passed through correctly to the create command. Workaround: use the tk CLI directly.
