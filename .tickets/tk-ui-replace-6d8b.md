---
id: tk-ui-replace-6d8b
stage: triage
status: in_progress
deps: []
links: []
created: 2026-02-28T20:19:56Z
type: feature
priority: 1
assignee: Steve Macbeth
tags: [tui]
---
# 'tk ui' Replace pipeline kanban with triage + inbox dashboard




Replace the full 6-column pipeline kanban with a two-pane dashboard optimized for the human workflow: (1) Triage pane — tickets at triage stage waiting to enter the pipeline, (2) Inbox pane — tickets needing human action (review gates, verify stage, rejected reviews, decisions). Full pipeline view accessible via P keybinding for when you want the big picture. Text search, priority cycling, create, and ticket detail drill-down carry over from current pipeline view.

## Test Results

`go test ./...` passes (no TUI-specific tests). Build compiles clean. **Manual testing required:** dashboard shows triage + inbox panes, tab switches focus, enter opens detail, P opens pipeline, esc returns from pipeline/detail, / search, t type filter, p priority, c create.
