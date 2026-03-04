---
id: core-replace-status-83c2
stage: triage
status: open
deps: [core-remove-status-17f0]
links: []
created: 2026-03-04T05:02:07Z
type: task
priority: 2
parent: remove-status-stage-7439
version: 2
---
# Core: Replace status-based filtering with stage

Replace all status-based filtering and sorting with stage-based equivalents.

## Design

Files: pkg/ticket/filter.go

- Replace ListOptions.Status (Status type) with ListOptions.Stage (Stage type)
- Update Filter(): check opts.Stage against t.Stage instead of status
- Delete statusOrder() function
- Add stageOrder() function: maps stage to pipeline position (triage=1, spec=2, design=3, implement=4, test=5, verify=6, done=7)
- Rename SortByStatusPriorityID → SortByStagePriorityID, use stageOrder() internally

## Acceptance Criteria

1. ListOptions has Stage field, no Status field
2. Filter() filters by stage
3. stageOrder() returns correct pipeline positions
4. Sort function uses stage ordering
