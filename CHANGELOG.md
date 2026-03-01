# Changelog

## [Unreleased]

### Fixed
- Ticket body accumulated extra blank lines on each save (parse→serialize round-trip)

## [2.4.0] - 2026-02-28

### Added
- TUI `v` key on verify tab advances ticket to next stage; `R` on review tab approves review
- TUI file watcher — auto-reloads tickets when `.tickets/` directory changes (fsnotify with 200ms debounce)
- TUI edit mode (`e`) — edit title, description, type, priority, assignee, stage, and add notes from the form view
- TUI `o` key as alias for `enter` to open ticket detail

### Changed
- TUI default view is now a single-pane inbox with tabbed filters: all, triage, verify, review
- Removed status-based list view and pipeline kanban as default — focused on human decision points
- Pipeline view now supports text search (`/`), priority cycling (`p`), and create (`c`)
- TUI detail view help bar uses consistent `(k)ey` format with `│` separators

### Fixed
- TUI form text fields wrap long text across multiple lines instead of scrolling horizontally off-screen
- TUI form text fields overflowed past terminal width — now truncated with cursor-aware viewport, left/right arrow movement, and home/end support
- MCP `ticket_create` didn't set `created` timestamp — tickets created via MCP had zero-value dates
- TUI list view ID column truncated slug-based IDs — column width now computed dynamically from visible tickets
- TUI pipeline view missing `c` keybinding for create — now matches list view behavior

## [2.3.0] - 2026-02-28

### Fixed
- TUI create form failed with "ticket ID is required" — `handleCreateTicket` was missing `GenerateID()` and `Created` timestamp

## [2.2.0] - 2026-02-27

### Added
- Encouraging messages on empty listing output — ls, ready, blocked, inbox, closed, pipeline, and next show a random message from a pool of 20 when results are empty. `--json` returns `[]`.

### Fixed
- MCP `ticket_create` failed with "ticket ID is required" — handler was missing ID generation, status, and stage initialization
- Notes with `**bold**` markdown lines were split into multiple notes during parsing — `parseNotes` now validates timestamp before flushing
- MCP `ticket_edit` silently dropped description, design, and acceptance fields — handler now uses `UpdateSection` to persist body fields
- MCP gate checks required body sections unreachable via `ticket_edit` — added `test_results` field and exposed `## Test Results` in show output

## [2.1.1] - 2026-02-26

### Fixed
- Homebrew install — use formula (`brews`) instead of cask (`homebrew_casks`) in GoReleaser config

## [2.1.0] - 2026-02-26

### Added
- **`--repo` global flag** — operate on any repo from anywhere (`tk ls --repo ~/code/other-project`). Walks up from the given path to find `.tickets/`, same as CWD resolution. Errors if no `.tickets/` found.
- **Stage pipeline system** — type-dependent stage pipelines replace flat status enum
  - 7 stages: triage → spec → design → implement → test → verify → done
  - Type-dependent pipelines: feature (7), bug (5), task (5), chore (3), epic (4)
- **Gate enforcement** — structural preconditions for stage transitions
  - Risk-scaled gates (low=advisory, normal=standard, high/critical=strict)
  - Mandatory code + impl review gates at implement → test
- **Review system** — ReviewState tracking (pending/approved/rejected) with ReviewRecord audit log
- **Pipeline workflow functions** — `Advance()`, `Skip()`, `SetReview()` in pkg/ticket
- **Stage propagation** — `PropagateStage()` for parent stage advancement based on children
- **Migration** — `MigrateTicket()`/`MigrateAll()` for status → stage conversion
  - Mapping: open→triage, in_progress→implement, needs_testing→test, closed→done
- **Inbox/next-action derivation** — `Inbox()`, `NextAction()`, `Projects()` for workflow visibility
- New Ticket struct fields: Stage, Review, Risk, Skipped, Conversations, Reviews
- Review Log section parsing and serialization in ticket markdown format
- `ValidateStageForType()`, `ValidateGates()` validation functions
- Pipeline helpers: `NextStage()`, `PrevStage()`, `HasStage()`, `StageIndex()`, `IsFinalStage()`
- **CLI commands:** `advance`, `skip`, `review`, `log`, `pipeline`, `inbox`, `next`, `migrate`
- **Edit flags:** `--stage`, `--review`, `--risk` for direct field editing
- **ls --group-by=pipeline** groups tickets by pipeline stage
- **Backward compatibility:** `start`/`close`/`reopen` map to stage equivalents with hint
- **MCP tools:** `ticket_advance`, `ticket_review`, `ticket_skip`, `ticket_migrate`, `ticket_inbox`
- New tickets default to `stage: triage` on creation
- Integration tests for all pipeline commands (188 assertions total)

### Changed
- **Human-readable ticket IDs** — IDs now use up to 3 meaningful words from the title instead of directory-name prefix (e.g., `fix-login-page-fe32` instead of `tic-fe32`). Existing tickets keep their IDs unchanged.
- `GenerateID()` now requires a title argument; stop words (articles, prepositions, etc.) are stripped from the slug
- `Store.Create()` returns an explicit duplicate error instead of relying on hash collision retry
- `ls` defaults to workflow grouping (In Progress / Ready / Blocked). Use `--flat` for the old flat list.
- `ls` shows dep count (`(2 deps)`) instead of full dep ID list (`<- [t-1234, t-5678]`)
- Ticket validation accepts either `status` (legacy) or `stage` (pipeline) — dual support for migration
- format.go writes stage/review/risk/skipped/conversations fields when present
- `show` checks both status and stage for blocker/blocking display
- `ls` excludes `stage: done` tickets from default view
- `printRow` shows stage when available, falls back to status
- Help text updated with pipeline commands and options
- MCP `toJSON` includes stage/review/risk/skipped/conversations/reviews fields

## [2.0.0] - 2026-02-23

Go rewrite. Full CLI parity with bash version plus new capabilities.
Both implementations remain supported and read/write the same ticket format.

### Added
- **Go binary** — cross-platform, single binary distribution via Homebrew and AUR
- **TUI** (`tk ui`) — interactive ticket browser with list/detail views, inline editing, ticket creation
- **MCP server** (`tk serve`) — stdio MCP server for Claude Code integration
- `--json` global flag on all output commands
- `--version` / `-v` flag (version injected at build time via GoReleaser)
- `stats` command — project health dashboard (status/type/priority breakdowns, open ticket age)
- `timeline` command — bar chart of tickets closed by week with `--weeks=N` flag
- `move` command — move tickets between repos with `--recursive` for full subtree moves
- `--group-by` flag for `ls` (workflow, type, status, priority) with `--group` shorthand
- `--note` flag for `edit` as alias for `add-note`
- `--design`, `--acceptance` flags support multiline text (bash awk limitation fixed)
- GoReleaser config for darwin/linux arm64/amd64 builds
- Comprehensive test suite (144 assertions)

### Changed
- ID generation uses nanosecond timestamps + atomic counter (eliminates rapid-create collisions)
- `create` retries with new ID on collision (up to 5 attempts)

### Fixed
- `ls --parent` now correctly filters to children only
- Multiline `--design` and `--acceptance` flags work correctly (bash awk limitation)
- ID collisions when creating multiple tickets per second

## [Unreleased - bash]

### Added
- `list` alias for `ls` command
- `needs_testing` status
- `-s, --status` flag for `edit` command to change ticket status
- Hierarchy gating: `ready` only shows tickets whose parent is `in_progress`
- `--open` flag for `ready` to bypass hierarchy checks
- Status propagation: `needs_testing`/`closed` auto-bubble up parent chain
- `workflow` command outputs guide for LLM context
- `-t, --type` filter flag for `ls` command
- Interactive prompts when `tk create` is run with no arguments
- Support `TICKETS_DIR` environment variable for custom tickets directory location
- `dep cycle` command to detect dependency cycles in open tickets
- `add-note` command for appending timestamped notes to tickets
- `-a, --assignee` filter flag for `ls`, `ready`, `blocked`, and `closed` commands
- `--tags` flag for `create` command to add comma-separated tags
- `-T, --tag` filter flag for `ls`, `ready`, `blocked`, and `closed` commands
- `-P, --priority` filter flag for `ls` command
- `delete` command to remove ticket files

### Changed
- `create` command now displays full ticket details on success instead of just the ID
- `edit` command now uses CLI flags instead of opening $EDITOR

### Removed
- `start`, `testing`, `close`, `reopen`, `status` commands (use `edit -s` instead)

### Fixed
- `update_yaml_field` now works on BSD/macOS (was using GNU sed syntax)

## [0.2.0] - 2026-01-04

### Added
- `--parent` flag for `create` command to set parent ticket
- `link`/`unlink` commands for symmetric ticket relationships
- `show` command displays parent title and linked tickets
- `migrate-beads` now imports parent-child and related dependencies

## [0.1.1] - 2026-01-02

### Fixed
- `edit` command no longer hangs when run in non-TTY environments

## [0.1.0] - 2026-01-02

Initial release.
