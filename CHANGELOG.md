# Changelog

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
