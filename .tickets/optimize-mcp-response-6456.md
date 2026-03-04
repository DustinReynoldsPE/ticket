---
id: optimize-mcp-response-6456
stage: done
status: in_progress
deps: []
links: []
created: 2026-03-03T15:31:38Z
type: feature
priority: 1
tags: [mcp, optimization]
skipped: [spec, design, test, verify]
version: 4
---
# Optimize MCP response payloads for context compression

MCP tool responses return full ticketJSON for every operation — mutations, lists, inbox. This wastes agent context tokens on data the agent doesn't need. Four changes: (1) switch json.MarshalIndent to json.Marshal (~15-20% savings), (2) add ticketSummaryJSON for list/ready/blocked/inbox responses (~60-80% savings on list calls), (3) return slim confirmation for mutation responses (~70-90% savings on writes), (4) add limit/offset pagination to ticket_list.

## Acceptance Criteria

- [ ] jsonResult uses compact json.Marshal instead of MarshalIndent
- [ ] ticket_list, ticket_ready, ticket_blocked return summary projections (id, title, status, stage, priority, type, assignee, parent)
- [ ] ticket_inbox returns summary projection with action/detail
- [ ] Mutation tools (create, edit, add_note, dep, link, advance, review, skip) return slim response (id plus changed fields)
- [ ] ticket_list accepts optional limit parameter (default 50)
- [ ] All existing MCP tests pass
- [ ] New tests cover summary vs full response formats

## Test Results

- [x] jsonResult uses compact json.Marshal instead of MarshalIndent\n- [x] ticket_list, ticket_ready, ticket_blocked return summary projections\n- [x] ticket_inbox returns summary projection with action/detail\n- [x] Mutation tools return slim summary response\n- [x] ticket_list accepts optional limit parameter (default 50)\n- [x] All 11 existing MCP tests pass\n- [x] 4 new tests: TestListReturnsSummary, TestListLimit, TestMutationReturnsSummary, TestShowReturnsFull
