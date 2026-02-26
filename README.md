# ticket

A git-backed issue tracker for AI agents. Rooted in the Unix Philosophy, `tk` is inspired by Joe Armstrong's [Minimal Viable Program](https://joearms.github.io/published/2014-06-25-minimal-viable-program.html) with additional quality of life features for managing and querying against complex issue dependency graphs.

Tickets are markdown files with YAML frontmatter in `.tickets/`. This allows AI agents to easily search them for relevant content without dumping ten thousand character JSONL lines into their context window.

Using ticket IDs as file names also allows IDEs to quickly navigate to the ticket. For example, you might run `git log` in your terminal and see something like:

```
nw-5c46: add SSE connection management
```

VS Code allows you to Ctrl+Click or Cmd+Click the ID and jump directly to the file to read the details.

## Install

There are two implementations: a Go binary and the original bash script. Both are fully compatible — they read and write the same ticket format.

### Go (recommended)

Requires Go 1.21+.

```bash
git clone https://github.com/EnderRealm/ticket.git
cd ticket && go build -o ~/.local/bin/tk .
```

The Go version includes everything in the bash version plus:
- **Stage pipeline** — type-dependent stage pipelines (triage → spec → design → implement → test → verify → done)
- **Pipeline commands** — `advance`, `skip`, `review`, `log`, `pipeline`, `inbox`, `next`, `migrate`
- **Gate enforcement** — structural preconditions for stage transitions, risk-scaled
- `tk ui` — interactive TUI with list view and pipeline kanban view
- `tk serve` — MCP server for Claude Code integration
- `--json` flag on all commands
- `--repo <path>` — operate on any repo from anywhere

### Bash (deprecated)

The original bash script still works for basic operations but does not support the stage pipeline system. New development targets the Go version only.

```bash
git clone https://github.com/EnderRealm/ticket.git
cd ticket && ln -s "$PWD/ticket" ~/.local/bin/tk
```

## Configuration

Set `TICKETS_DIR` to store tickets in a custom location (default: `.tickets`):

```bash
export TICKETS_DIR=".tasks"
tk create "my ticket"
```

Use `--repo` to operate on a different repo from anywhere:

```bash
tk ls --repo ~/code/other-project
tk show fix-auth --repo ~/code/other-project
```

## Agent Setup

Add this line to your `CLAUDE.md` or `AGENTS.md`:

```
This project uses a CLI ticket system for task management. Run `tk help` when you need to use it.
```

Claude Opus picks it up naturally from there. Other models may need additional guidance.

## Usage

Run `tk help` for full command reference. Key commands:

```
Viewing:
  show <id>                  Display ticket details
  ls [filters]               List tickets (default: open only)
  ready [filters]            Unblocked tickets ready to work on
  blocked [filters]          Tickets with unresolved deps
  closed [--limit=N]         Recently closed tickets

Creating & Editing:
  create [title] [options]   Create ticket
  edit <id> [options]        Update ticket fields
  add-note <id> [text]       Append timestamped note
  delete <id> [id...]        Delete ticket(s)

Pipeline:
  advance <id> [--to stage]  Advance to next pipeline stage
  skip <id> --to <stage>     Skip ahead with --reason justification
  review <id> --approve      Record review verdict (--approve or --reject)
  log <id>                   Show stage/review history
  pipeline [--stage X]       Show tickets grouped by pipeline stage
  inbox                      Show tickets needing human attention
  next                       Per-project next actions
  migrate [--dry-run]        Migrate legacy tickets to stage pipeline

Dependencies & Links:
  dep <id> <dep-id>          Add dependency
  undep <id> <dep-id>        Remove dependency
  link <id> <id>             Link tickets (symmetric)

Query:
  query [jq-filter]          Output tickets as JSONL

Analytics:
  stats                      Project health dashboard
  timeline [--weeks=N]       Tickets closed by week

Interactive:
  ui                         Terminal UI (list + pipeline kanban view)
  serve                      MCP server for AI agent integration
```

### Pipeline Stages

Tickets progress through type-dependent stage pipelines:

| Type | Pipeline |
|------|----------|
| feature | triage → spec → design → implement → test → verify → done |
| bug | triage → implement → test → verify → done |
| task | triage → implement → test → verify → done |
| chore | triage → implement → done |
| epic | triage → spec → design → done |

Gate checks enforce preconditions at stage transitions (e.g., acceptance criteria before spec → design, review approval before implement → test). Gates scale by risk level: low (advisory), normal (standard), high/critical (strict).

### Filter Flags

```
--status X        open | in_progress | needs_testing | closed
-t, --type X      bug | feature | task | epic | chore
-P, --priority X  0 (critical) through 4 (backlog)
-a, --assignee X  Filter by assignee
-T, --tag X       Filter by tag
--parent X        Children of ticket X
--group-by X      Group by: workflow | pipeline | type | status | priority
```

Partial ID matching: `tk show 5c4` matches `nw-5c46`.

## License

MIT
