---
id: mcp-replace-status-5712
stage: triage
status: open
deps: [core-remove-status-17f0]
links: []
created: 2026-03-04T05:02:31Z
type: task
priority: 2
parent: remove-status-stage-7439
version: 2
---
# MCP: Replace status with stage in tool parameters

Remove status from all MCP tool definitions and responses, replace with stage.

## Design

Files: internal/mcp/mcp.go

listArgs:
- Remove Status field
- Add Stage field: json:"stage,omitempty" jsonschema:"filter by stage: triage, spec, design, implement, test, verify, done"
- Update filtering logic: if args.Stage set, opts.Stage = ticket.Stage(args.Stage); else exclude StageDone

createArgs:
- Remove Status assignment in handler (only set Stage: StageTriage)

editArgs:
- Remove Status field and its processing block
- Add Stage field if not already present

ticketSummaryJSON / ticketFullJSON:
- Remove Status field from JSON output structs
- Keep Stage field

reviewJSON: no changes needed (already stage-only)

## Acceptance Criteria

1. No status in any MCP tool parameter
2. No status in any MCP JSON response
3. List filtering uses stage
4. Create only sets stage: triage
