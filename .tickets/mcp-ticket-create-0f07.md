---
id: mcp-ticket-create-0f07
stage: implement
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

## Notes

**2026-02-27T07:49:47Z**

Root cause: MCP registerCreate built the Ticket struct without setting ID, Status, or Stage. Validate() fails on empty ID before store.Create can proceed. Fix: generate ID via GenerateID(title), set Status=open, Stage=triage — matching what cmd/create.go does.
