---
id: t-dcbb
status: open
deps: [t-dde3]
links: []
created: 2026-02-22T00:58:07Z
type: task
priority: 1
assignee: Steve Macbeth
parent: t-0f08
tags: [phase-1]
---
# Filter and sort logic

Implement ticket filtering and sorting used by ls, ready, blocked, closed commands.

## Design

Files: pkg/ticket/filter.go
Approach:
- Filter(tickets []*Ticket, opts ListOptions) []*Ticket: apply all filters
- Individual filter predicates: ByStatus, ByType, ByPriority, ByAssignee, ByTag, ByParent
- Sort functions: ByPriorityThenTitle (default for ls/ready/blocked), ByModTime (for closed)
- ListOptions carries all filter params, shared across commands
- Composable: filters chain, sort applies after filter

## Acceptance Criteria

Filters correctly narrow ticket lists. Sort order matches current bash output.

