---
id: mcp-ticket-add-310d
stage: triage
status: open
deps: []
links: []
created: 2026-02-27T07:42:31Z
type: bug
priority: 2
assignee: Steve Macbeth
tags: [mcp]
---
# MCP ticket_add_note produces duplicate and garbled notes


When adding notes via the MCP ticket_add_note tool, notes appear multiple times in the ticket file with garbled formatting — fields concatenated without newlines, missing line breaks between sections. See show-encouraging-message-f072 Notes section for example: 4 duplicate triage entries and 3 duplicate spec entries with mangled text.
