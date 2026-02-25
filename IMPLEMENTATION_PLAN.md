# Implementation Plan

Staged, incremental plan for the agentic workflow redesign. Each phase is independently testable and deployable. Phases 1-3 are in the `tk` repo. Phases 4-7 are in the new `forge` repo.

**Guiding principle:** MCP-first. Every new capability is implemented once in `pkg/ticket/`, exposed through both CLI (`cmd/*.go`) and MCP (`internal/mcp/mcp.go`). The MCP server becomes the primary integration surface for all agent interactions. The CLI remains for human use and debugging.

---

## Phase 1: Stage Pipeline Core

**Repo:** `tk`
**Goal:** Replace the `status` enum with `stage` + type-dependent pipelines in the core library. No CLI or MCP changes yet — just the data model and logic.

### 1.1 New types in `pkg/ticket/ticket.go`

```go
type Stage string

const (
    StageTriage    Stage = "triage"
    StageSpec      Stage = "spec"
    StageDesign    Stage = "design"
    StageImplement Stage = "implement"
    StageTest      Stage = "test"
    StageVerify    Stage = "verify"
    StageDone      Stage = "done"
)

type ReviewState string

const (
    ReviewNone     ReviewState = ""          // no review in progress
    ReviewPending  ReviewState = "pending"   // awaiting review
    ReviewApproved ReviewState = "approved"  // review passed
    ReviewRejected ReviewState = "rejected"  // review failed, needs rework
)

// ReviewRecord tracks a single review verdict with context.
// Stored as structured notes in the ticket body (## Review Log section).
type ReviewRecord struct {
    Timestamp time.Time
    Stage     Stage       // which stage was being reviewed
    Kind      string      // "spec", "design", "impl", "code", "verify"
    Actor     string      // "human:steve", "agent:code-reviewer"
    State     ReviewState // approved or rejected
    Comment   string
}

// WaitingOn describes what a ticket needs to move forward.
type WaitingOn struct {
    Type   string  // "human-review", "agent-review", "human-input", "agent-work", "dependency", "nothing"
    Detail string  // e.g. "design review needed", "blocked by t-5a3f"
    Since  time.Time
}

```

### 1.2 Type-dependent pipeline definitions in new file `pkg/ticket/pipeline.go`

```go
// Pipeline defines the ordered stages for a ticket type.
var Pipelines = map[TicketType][]Stage{
    TypeFeature: {StageTriage, StageSpec, StageDesign, StageImplement, StageTest, StageVerify, StageDone},
    TypeBug:     {StageTriage, StageImplement, StageTest, StageVerify, StageDone},
    TypeChore:   {StageTriage, StageImplement, StageDone},
    TypeEpic:    {StageTriage, StageSpec, StageDesign, StageDone},
    TypeTask:    {StageTriage, StageImplement, StageTest, StageVerify, StageDone}, // same as bug
}

// NextStage returns the next stage for a ticket's type, or error if at end.
func NextStage(typ TicketType, current Stage) (Stage, error)

// PrevStage returns the previous stage.
func PrevStage(typ TicketType, current Stage) (Stage, error)

// HasStage checks whether a ticket type includes a specific stage.
func HasStage(typ TicketType, stage Stage) bool

// StageIndex returns the position of a stage in a type's pipeline (-1 if not present).
func StageIndex(typ TicketType, stage Stage) int
```

### 1.3 Gate definitions in `pkg/ticket/gates.go`

```go
// GateCheck represents a precondition for advancing past a stage.
type GateCheck struct {
    Stage       Stage
    Description string
    Check       func(t *Ticket, store *FileStore) error
}

// Gates returns the gate checks for advancing FROM the given stage.
func Gates(typ TicketType, from Stage) []GateCheck
```

Gate rules (from REDESIGN.md decisions):
- `triage → spec/implement`: description exists
- `spec → design`: acceptance criteria section exists, review=approved
- `design → implement`: design + plan sections exist, review=approved
- `design → done` (epic): design exists, review=approved
- `implement → test`: **mandatory** code-review=approved AND impl-review=approved
- `implement → done` (chore): advisory review surfaced (logged, not blocking)
- `test → verify`: test results recorded, all pass
- `verify → done`: review=approved

### 1.4 Extended Ticket struct in `pkg/ticket/ticket.go`

```go
type Ticket struct {
    ID          string      `yaml:"id"`
    Stage       Stage       `yaml:"stage"`           // NEW: replaces Status
    Status      Status      `yaml:"status,omitempty"` // DEPRECATED: kept for migration
    Review      ReviewState `yaml:"review,omitempty"` // NEW
    Type        TicketType  `yaml:"type"`
    Priority    int         `yaml:"priority"`
    Assignee    string      `yaml:"assignee,omitempty"`
    Parent      string      `yaml:"parent,omitempty"`
    Deps        []string    `yaml:"deps,flow"`
    Links       []string    `yaml:"links,flow"`
    Tags        []string    `yaml:"tags,omitempty,flow"`
    ExternalRef string      `yaml:"external-ref,omitempty"`
    Created     time.Time   `yaml:"created"`
    Skipped     []Stage     `yaml:"skipped,omitempty,flow"` // NEW
    Conversations []string  `yaml:"conversations,omitempty,flow"` // NEW
    WaitOn      string      `yaml:"wait-on,omitempty"` // NEW: what's blocking progress

    // Parsed from markdown, not stored in frontmatter.
    Title   string         `yaml:"-"`
    Body    string         `yaml:"-"`
    Notes   []Note         `yaml:"-"`
    Reviews []ReviewRecord `yaml:"-"` // NEW: parsed from ## Review Log section
}
```

### 1.5 Advance logic in `pkg/ticket/workflow.go`

```go
// AdvanceResult holds the outcome of an advance attempt.
type AdvanceResult struct {
    From        Stage
    To          Stage
    Skipped     []Stage       // stages skipped (if --skip-to used)
    GateFailed  []GateCheck   // gates that didn't pass (for advisory display)
    Propagation []StatusChange // parent status changes triggered
}

// Advance moves a ticket to its next pipeline stage.
// Checks gates. Returns error if mandatory gate fails.
func Advance(store *FileStore, id string, opts AdvanceOptions) (*AdvanceResult, error)

// AdvanceOptions controls advance behavior.
type AdvanceOptions struct {
    SkipTo  Stage  // jump to this stage (skipping intermediates)
    Reason  string // required if SkipTo is set
    Force   bool   // bypass advisory gates (not mandatory ones)
}

// Review records a review verdict on a ticket.
func SetReview(store *FileStore, id string, state ReviewState, comment string, actor string) error

// Skip moves a ticket to a non-adjacent stage with audit trail.
func Skip(store *FileStore, id string, to Stage, reason string) error
```

### 1.6 Propagation updates in `pkg/ticket/workflow.go`

Update `PropagateStatus` to work with stages:
- All children at `done` → parent advances to `done`
- All children at `test` or later → parent advances to `test` (if applicable to type)
- Rename to `PropagateStage` (keep `PropagateStatus` as deprecated wrapper)

### 1.7 Migration logic in `pkg/ticket/migrate.go`

```go
// MigrateTicket converts a status-based ticket to stage-based.
func MigrateTicket(t *Ticket) {
    if t.Stage != "" { return } // already migrated
    switch t.Status {
    case StatusOpen:         t.Stage = StageTriage
    case StatusInProgress:   t.Stage = StageImplement
    case StatusNeedsTesting: t.Stage = StageTest
    case StatusClosed:       t.Stage = StageDone
    }
    t.Status = "" // clear deprecated field
}

// NeedsMigration checks if a ticket uses the old status field.
func NeedsMigration(t *Ticket) bool
```

### 1.8 Format updates in `pkg/ticket/format.go`

- `Serialize`: Write `stage` instead of `status`. Write `review`, `conversations`, `skipped` if non-empty. If `Status` is still set (migration compat), write `status` field too.
- `Parse`: Read both `stage` and `status`. If only `status` present, auto-migrate in memory (don't write back — that's `tk migrate`'s job).
- New body sections: `## Review Log`, `## Implementation Plan`, `## Test Results`

### 1.9 Validation updates

- `Validate()` accepts either `stage` or `status` (not neither)
- `ValidateStage()` checks stage is valid for ticket type
- `ValidateGates()` checks all gate preconditions without advancing

### 1.10 Inbox + Next Action derivation in `pkg/ticket/inbox.go`

The unified inbox and "what's next" views are **derived state** — they're computed from ticket fields, not stored separately. This keeps the source of truth in the ticket files and avoids sync problems.

```go
// ActionKind classifies what a ticket needs right now.
type ActionKind string

const (
    ActionHumanReview ActionKind = "human-review"  // review=pending, actor is human
    ActionAgentReview ActionKind = "agent-review"   // review=pending, actor is agent
    ActionHumanInput  ActionKind = "human-input"    // conversational stage (triage, spec, verify)
    ActionAgentWork   ActionKind = "agent-work"     // autonomous stage (design, implement, test)
    ActionBlocked     ActionKind = "blocked"        // unresolved dependency
    ActionReady       ActionKind = "ready"          // can advance, nothing blocking
)

// InboxItem represents one thing that needs human attention.
type InboxItem struct {
    Ticket     *Ticket
    Action     ActionKind
    Detail     string      // human-readable: "design review needed", "spec: build acceptance criteria"
    Since      time.Time   // when this item entered the inbox (stage change timestamp)
    Repo       string      // which repo this ticket lives in (workspace mode, "" for single-repo)
    Project    string      // parent epic title (or "" if standalone)
    ProjectID  string      // parent epic ID (or "")
    Priority   int         // inherited: max(own priority, parent priority)
}

// Inbox returns all tickets that need human attention, sorted by priority
// then age (oldest first within same priority).
// This is the unified inbox — the single view of "what needs me."
func Inbox(store *FileStore) ([]InboxItem, error)

// NextAction returns the computed next action for a single ticket.
func NextAction(t *Ticket, store *FileStore) InboxItem
```

**What lands in the inbox** (items needing human action):

| Condition | ActionKind | Example |
|-----------|-----------|---------|
| Stage is conversational + not done | `human-input` | Ticket at `spec`, needs human to build AC |
| `review: pending` + last review actor is `human:*` | `human-review` | Design review awaiting human approval |
| `review: pending` + no actor hint → assume human | `human-review` | Review requested, nobody assigned |
| `review: rejected` by agent, needs human judgment | `human-review` | Agent rejected but human may override |
| Gate failed on advance attempt (advisory) | `human-review` | Gate flagged an issue, human decides |

**What does NOT land in the inbox** (handled by agents/automation):

| Condition | ActionKind | Where it shows |
|-----------|-----------|---------------|
| `review: pending` + actor is `agent:*` | `agent-review` | Orchestrator queue |
| Stage is autonomous + ready for agent | `agent-work` | Orchestrator queue |
| All deps resolved, no review needed | `ready` | "What's Next" panel |
| Unresolved deps | `blocked` | Blocked panel |

**Deriving "conversational" vs "autonomous" stage:**

```go
// ConversationalStages are stages that require human interaction.
// Everything else is autonomous (agent-driven).
var ConversationalStages = map[Stage]bool{
    StageTriage: true,
    StageSpec:   true,
    StageVerify: true,
}

// IsConversational returns true if the stage requires human interaction.
func IsConversational(stage Stage) bool
```

**Project grouping for "What's Next":**

```go
// ProjectSummary aggregates ticket state across a project (epic + children).
// In workspace mode, a project may span multiple repos.
type ProjectSummary struct {
    Epic          *Ticket       // the parent epic (nil for standalone tickets)
    Tickets       []*Ticket     // all tickets in this project (may span repos)
    Repos         []string      // which repos contribute tickets to this project
    InboxCount    int           // how many items need human attention
    BlockedCount  int           // how many are blocked
    ActiveCount   int           // how many are in progress (not triage, not done)
    Progress      float64       // 0.0-1.0: fraction of pipeline stages completed across all tickets
    NextActions   []InboxItem   // what needs doing, sorted by priority
    StageBreakdown map[Stage]int // how many tickets at each stage
}

// Projects returns all active projects (epics with children + standalone tickets),
// sorted by priority then progress (most complete first — finish what you started).
func Projects(store *FileStore) ([]ProjectSummary, error)
```

**Sort order for "What's Next":** Most-complete-first within same priority. A ticket at `test` stage is closer to done than one at `triage` — prioritize finishing it. This prevents the trap of always starting new work.

### 1.11 Review log parsing in `pkg/ticket/format.go`

Reviews are stored in the ticket body as a structured `## Review Log` section:

```markdown
## Review Log

**2026-02-25T14:30:00Z** [design] agent:design-reviewer → approved
Codebase paths verified. API patterns consistent with existing handlers.

**2026-02-25T15:00:00Z** [design] human:steve → approved
LGTM, proceed to implementation.

**2026-02-25T18:45:00Z** [impl] agent:impl-reviewer → rejected
Acceptance criteria #3 (error handling) not covered. Missing validation in CreateHandler.
```

- `Parse()` extracts `ReviewRecord` structs from this section
- `Serialize()` writes them back in canonical format
- `SetReview()` appends to this section (doesn't overwrite)
- The `review` YAML field reflects the *latest* review state; the log has full history

This means the inbox can determine: "who is the review waiting on?" by looking at the review kind and the gate definition for the current stage. If the gate says `impl-review` and `code-review` must pass, and `impl-review` is `approved` but `code-review` hasn't happened yet, the inbox knows to show "code review pending."

### 1.12 MCP tools for inbox/next in Phase 2 (preview)

These tools get implemented in Phase 2, but the core logic is built here:

```go
// ticket_inbox: Returns items needing human attention.
// Filters: --project <epic-id>, --action <kind>, --assignee <name>
// Sort: priority, then age

// ticket_next: Returns per-project summary of what to do next.
// Filters: --project <epic-id>
// Returns: ProjectSummary[] with inbox items and progress

// ticket_dashboard: Single call that returns everything the dashboard home needs.
// Returns: { inbox: InboxItem[], projects: ProjectSummary[], stats: {...} }
// This avoids N+1 MCP calls from the dashboard.
```

### 1.13 Tests

- `pipeline_test.go`: Pipeline definitions, NextStage, HasStage for all types
- `gates_test.go`: Gate checks for every transition, mandatory vs advisory
- `workflow_test.go`: Update existing + add Advance, Skip, SetReview tests
- `migrate_test.go`: All status→stage mappings, idempotency, round-trip
- `format_test.go`: Update for new fields, backward compat with old format, review log round-trip
- `inbox_test.go`: Inbox derivation — conversational stages surface as human-input, pending reviews surface correctly, blocked tickets excluded from inbox, project grouping works, sort order respects priority then completeness

**Testable checkpoint:** All unit tests pass. `go test ./pkg/ticket/...` green. No CLI or MCP changes yet — this is purely library-level.

---

## Phase 2: CLI + MCP Parity

**Repo:** `tk`
**Goal:** Expose all new pipeline features through both CLI and MCP. Every command below gets both a `cmd/*.go` and an `internal/mcp/` registration.

### 2.1 New CLI commands

| Command | Description |
|---------|-------------|
| `tk advance <id> [--to <stage>] [--reason "..."]` | Move ticket to next stage (or skip to named stage) |
| `tk review <id> --approve\|--reject [--comment "..."] [--actor "..."]` | Record review verdict |
| `tk skip <id> --to <stage> --reason "..."` | Skip stages with audit trail (alias for advance --to) |
| `tk log <id>` | Show full stage transition history (from Notes) |
| `tk pipeline [--stage <stage>] [--type <type>]` | List tickets grouped by pipeline stage |
| `tk migrate [--dry-run]` | Rewrite all tickets: status→stage |
| `tk inbox [--project <epic-id>] [--assignee <name>]` | Show items waiting on human action |
| `tk next [--project <epic-id>]` | Show per-project next actions and progress |

### 2.2 New MCP tools

| Tool | Description |
|------|-------------|
| `ticket_advance` | Advance a ticket through its pipeline |
| `ticket_review` | Record a review verdict |
| `ticket_skip` | Skip to a non-adjacent stage |
| `ticket_log` | Get stage transition history |
| `ticket_pipeline` | Get tickets grouped by pipeline stage |
| `ticket_migrate` | Run migration (status→stage) |
| `ticket_inbox` | Items needing human attention (for dashboard unified inbox) |
| `ticket_next` | Per-project summaries with next actions (for dashboard what's-next) |
| `ticket_dashboard` | Combined inbox + projects + stats in one call (avoids N+1) |

### 2.3 Updated CLI commands

| Command | Change |
|---------|--------|
| `tk ls` | Add `--stage` filter. Default grouping: pipeline view. Support `--group-by stage` |
| `tk ready` | Stage-aware: shows tickets ready for their current stage's next action |
| `tk show` | Display stage, review state, pipeline position, conversations, review log |
| `tk create` | New tickets start at `triage` stage. Accept `--stage` for testing |
| `tk edit` | Add `--stage`, `--review`, `--add-conversation` flags |
| `tk status` | DEPRECATED: prints warning suggesting `tk advance` |

### 2.4 Updated MCP tools

| Tool | Change |
|------|--------|
| `ticket_list` | Add `stage` filter, return stage/review in JSON |
| `ticket_ready` | Stage-aware readiness |
| `ticket_show` | Return stage, review, conversations, pipeline position |
| `ticket_create` | Set stage=triage, return stage in response |
| `ticket_edit` | Support stage, review, conversation fields |
| `ticket_workflow` | Updated guide with pipeline stages and type-dependent info |

### 2.5 MCP tool descriptions

Update all tool descriptions to reference stages instead of statuses. The `ticket_workflow` tool becomes the primary reference for agents to understand the pipeline.

New `ticket_workflow` output:
```
Ticket Workflow Guide

Types and their pipelines:
  feature:  triage → spec → design → implement → test → verify → done
  bug:      triage → implement → test → verify → done
  chore:    triage → implement → done
  epic:     triage → spec → design → done
  task:     triage → implement → test → verify → done

Advancing tickets:
  Use ticket_advance to move a ticket to its next stage.
  Gates are checked automatically. Mandatory gates block advancement.
  Advisory gates surface feedback but don't block.

Reviews:
  Use ticket_review to approve or reject at any stage.
  Mandatory reviews: implement → test (code-review + impl-review must pass)
  Advisory reviews: all other stage transitions

Stage skipping:
  Use ticket_skip for edge cases (bug that needs design, chore that needs testing).
  Requires a reason for the audit trail.
```

### 2.6 Backward compatibility

- `tk status <id> <status>` still works but prints deprecation warning and maps to appropriate `tk advance` behavior
- `ticket_edit` with `status` field still works, maps internally to stage
- `tk ls --status open` still works, maps to appropriate stage filter

### 2.7 Tests

- Update `test-suite.sh` with new commands
- Add integration tests for advance, review, skip, pipeline
- Add backward compat tests (old commands still work with warnings)
- Test MCP tools via Go test harness (call tool handlers directly)

### 2.8 Workspace mode (multi-repo)

Tickets stay in their repos — `tk/.tickets/`, `webapp/.tickets/`, `api/.tickets/`. No central store, no sync. Workspace mode lets `tk` query across all of them through a single CLI or MCP connection.

**Config file:** `~/.config/tk/workspace.yaml` (or `forge/.tk-workspace` for project-scoped):

```yaml
repos:
  - path: ~/code/tk
    name: tk
  - path: ~/code/webapp
    name: webapp
  - path: ~/code/api
    name: api
```

**How it works:**

```go
// pkg/ticket/workspace.go

// Workspace holds multiple FileStores, one per repo.
type Workspace struct {
    Repos []RepoStore
}

type RepoStore struct {
    Name  string      // short name for qualification: "tk", "webapp"
    Store *FileStore  // points to that repo's .tickets/ dir
}

// List returns tickets from all repos. Each ticket's ID is qualified
// with repo name: "tk:t-5a3f", "webapp:t-8b2c".
func (w *Workspace) List() ([]*Ticket, error)

// Get resolves a qualified or unqualified ID across all repos.
// Unqualified IDs work if unambiguous across repos.
func (w *Workspace) Get(id string) (*Ticket, error)

// Inbox, Projects, etc. all aggregate across repos.
func (w *Workspace) Inbox() ([]InboxItem, error)
func (w *Workspace) Projects() ([]ProjectSummary, error)
```

**Qualified IDs:** When displaying tickets from multiple repos, IDs get a repo prefix: `tk:t-5a3f`. When there's no ambiguity, the bare ID `t-5a3f` still works. Writes route to the correct repo's `.tickets/` directory based on the qualified ID.

**CLI usage:**
```bash
# Single-repo (default, unchanged behavior):
cd ~/code/tk && tk inbox         # reads only tk/.tickets/

# Workspace mode:
tk inbox --workspace ~/.config/tk/workspace.yaml   # reads all repos
tk inbox -w                                         # uses default workspace config

# Or set env var:
export TK_WORKSPACE=~/.config/tk/workspace.yaml
tk inbox                                            # reads all repos
```

**MCP usage:**
```bash
# Single-repo MCP server (unchanged):
tk serve                            # serves current repo's .tickets/

# Workspace MCP server (one connection, all repos):
tk serve --workspace ~/.config/tk/workspace.yaml

# The dashboard connects to this one server and sees everything.
```

**Cross-repo dependencies:** A ticket in `webapp` can depend on a ticket in `tk`:
```yaml
deps: [tk:t-5a3f]
```
The workspace resolves these across repos. `ReadyTickets()` and `BlockedTickets()` work across repo boundaries.

**Implementation in `pkg/ticket/`:**
- `workspace.go`: `Workspace` struct, multi-repo `List`/`Get`/`Inbox`/`Projects`
- All existing functions that take `*FileStore` gain workspace-aware variants that take `*Workspace` (or use an interface that both implement)
- `StoreInterface` interface: `List()`, `Get()`, `Update()`, `Create()`, `Delete()`, `Resolve()` — implemented by both `FileStore` (single repo) and `Workspace` (multi-repo)

**In `internal/mcp/mcp.go`:**
- `NewServer` accepts either `*FileStore` or `*Workspace`
- When in workspace mode, `ticket_list` returns qualified IDs, `ticket_inbox` aggregates across repos
- `ticket_create` requires a `repo` parameter in workspace mode to know which repo gets the ticket

**Testable checkpoint:** `tk inbox -w` shows items from multiple repos. `tk serve --workspace ...` exposes all repos via one MCP connection. Cross-repo deps resolve correctly. Qualified IDs work in all commands.

---

## Phase 3: TUI + Polish

**Repo:** `tk`
**Goal:** Update the TUI for pipeline view. Cut a release with migration support.

### 3.1 TUI updates (`internal/tui/`)

- Pipeline view: tickets grouped in columns by stage (like kanban but following pipeline order)
- Stage advancement: select ticket, press key to advance (with gate check feedback)
- Review actions: approve/reject from TUI
- Filter by type to see type-specific pipeline
- Color-coding: stage colors, review state indicators

### 3.2 Release prep

- Update CHANGELOG.md with all new features
- Update README.md with new commands and pipeline documentation
- Update `cmd_help()` / help text for all modified commands
- Tag release with migration support (dual status+stage)
- Follow-up release that drops status support

### 3.3 Bash script

The old `ticket` bash script: **freeze it.** No new features. Add a deprecation notice pointing to the Go binary. It continues working for anyone who hasn't upgraded, but all development happens in Go.

**Testable checkpoint:** TUI shows pipeline view. `tk` release published. Homebrew/AUR updated. `tk migrate` works on real ticket sets.

---

## Phase 4: Forge Repository Setup

**Repo:** NEW `forge`
**Goal:** Create the consolidated repo structure. Move skills, dashboard, and learning data in. No new features yet — just consolidation.

### 4.1 Repo creation

```
forge/
├── CLAUDE.md                # Forge-specific instructions
├── skills/                  # From powers (Claude Code skills)
│   ├── triage/SKILL.md
│   ├── spec/SKILL.md
│   ├── design/SKILL.md
│   ├── implement/SKILL.md
│   ├── review/SKILL.md
│   ├── test-ticket/SKILL.md
│   ├── investigate/SKILL.md
│   ├── brainstorm/SKILL.md
│   └── using-forge/SKILL.md
├── agents/                  # Agent definitions (.claude/agents/)
│   ├── spec-builder.md
│   ├── design-reviewer.md
│   ├── impl-reviewer.md
│   ├── code-reviewer.md
│   └── test-runner.md
├── hooks/                   # Claude Code hooks (from powers)
├── src/
│   ├── server/              # API + orchestrator (from manager)
│   └── client/              # Web dashboard (from manager)
├── scripts/                 # Nightly pipelines (from manager)
├── data/                    # Learnings (from ghost-data)
│   ├── sessions/
│   ├── patterns/
│   └── rollups/
└── package.json             # (or go.mod, depending on manager's stack)
```

### 4.2 Move strategy

1. Create `forge` repo with clean structure
2. Copy (not move) files from powers → `forge/skills/`, `forge/hooks/`
3. Copy files from manager → `forge/src/`
4. Copy files from learnings → `forge/data/`
5. Update all internal paths and imports
6. Verify everything builds and runs
7. Update powers/manager/learnings READMEs to point to forge
8. Archive old repos (don't delete — keep history accessible)

### 4.3 Claude Code plugin configuration

`forge/.claude-plugin/` or equivalent — register the skills directory so Claude Code discovers them.

**Testable checkpoint:** `forge` repo exists. All moved code builds. Skills load in Claude Code. Dashboard runs. Learning scripts run. Old repos archived with redirect notices.

---

## Phase 5: Stage-Specific Skills (MCP-First)

**Repo:** `forge`
**Goal:** Rewrite skills to use MCP tools instead of bash commands. Each pipeline stage gets a focused skill.

### 5.1 Skill architecture change

**Before (powers-style):** Skills contain bash blocks that shell out to `tk`:
```markdown
## Instructions
1. Run `tk create "..." --type feature`
2. Run `tk edit $ID --status in_progress`
```

**After (forge-style):** Skills describe intent and reference MCP tools:
```markdown
## Instructions
1. Use the `ticket_create` tool with type "feature"
2. Use the `ticket_advance` tool to move past triage
```

Claude Code calls the MCP tools directly — no bash intermediary.

### 5.2 Stage-specific skills

| Skill | Invocation | Stage | Mode | What it does |
|-------|-----------|-------|------|-------------|
| `/triage` | Conversational | triage | Interactive | Capture idea, ask clarifying questions, create ticket, set type/priority |
| `/spec` | Conversational | spec | Interactive | Build acceptance criteria (EARS format), scope definition, context gathering. End with focused decision summary in Notes |
| `/design` | Autonomous + review | design | Auto then interactive | Agent writes design + implementation plan. Design-review agent validates. Human reviews and approves |
| `/implement` | Autonomous | implement | Auto | Agent implements following the design. Impl-review + code-review agents run. Mandatory gates |
| `/review` | Autonomous | implement | Auto | Trigger review agents on a ticket (standalone, for re-review) |
| `/test-ticket` | Autonomous | test | Auto | Run test suite, record results, advance if pass |
| `/verify` | Conversational | verify | Interactive | Walk through acceptance criteria with human, approve/reject |
| `/investigate` | Any | any | Interactive | Debug/research (unchanged from current) |
| `/brainstorm` | Pre-triage | none | Interactive | Design sessions (unchanged from current, but now links to ticket via conversation) |

### 5.3 Skill template

Each skill follows this structure:
```markdown
---
name: <stage-name>
description: <one-line>
---

## Context
You are working on ticket $ARGUMENTS (or creating a new ticket).
The tk MCP server is available with tools: ticket_create, ticket_show, ticket_advance, ticket_review, etc.

## Stage: <stage>
This ticket is at the <stage> stage. Your job is: <focused mandate>.

## Instructions
<numbered steps using MCP tool names>

## Gate Requirements
Before advancing, ensure:
- <gate 1>
- <gate 2>

## On Completion
1. Use ticket_advance to move to the next stage
2. Write a decision summary to Notes (if conversational)
3. Record session ID via ticket_edit --add-conversation
```

### 5.4 Conversation tracking in skills

Conversational skills (triage, spec, verify) include this at the end:
```markdown
## Session Close
Before ending this conversation:
1. Write a 3-5 bullet decision summary to the ticket's Notes section using ticket_add_note
2. Record this session ID on the ticket using ticket_edit with the conversations field
```

**Testable checkpoint:** All skills work in Claude Code. `/triage` creates a ticket at triage stage via MCP. `/spec` builds acceptance criteria and advances via MCP. The full pipeline can be walked through manually using skills.

---

## Phase 6: Agent Definitions

**Repo:** `forge`
**Goal:** Define the focused agents that automate non-conversational stages.

### 6.1 Agent definitions (`forge/agents/` → `.claude/agents/`)

Each agent is a markdown file with YAML frontmatter defining its role, tools, and constraints.

| Agent | Trigger | Input | Output | Tools |
|-------|---------|-------|--------|-------|
| `spec-builder` | Ticket at triage, approved to advance | Triage description | Structured spec with EARS criteria | Read, Glob, Grep, WebSearch, ticket_* MCP |
| `design-reviewer` | Design written, needs review | Design section + codebase | READY or REVISE verdict + comments | Read, Glob, Grep, ticket_review |
| `impl-reviewer` | Implementation done | Code changes + acceptance criteria | APPROVED or REJECTED + checklist | Read, Glob, Grep, ticket_review |
| `code-reviewer` | Implementation done | Code diff | APPROVED or REJECTED + comments | Read, Glob, Grep, ticket_review |
| `test-runner` | Implementation reviewed | Test plan + codebase | Test results + pass/fail | Read, Bash (test only), ticket_advance |

### 6.2 Agent definition format

```markdown
---
name: design-reviewer
description: Validates implementation designs against the codebase
model: opus  # default, configurable
tools:
  - Read
  - Glob
  - Grep
  - mcp: tk  # Access to ticket MCP tools
---

## Role
You are a design review agent. Your job is to validate that an implementation design is feasible given the current codebase.

## Input
You will receive a ticket ID. Read the ticket using ticket_show.
Focus on the Design section and Implementation Plan.

## Checks
1. Do all referenced file paths exist? (Glob/Read to verify)
2. Are the proposed API patterns consistent with existing code? (Grep for similar patterns)
3. Does the implementation plan cover all acceptance criteria?
4. Are there obvious conflicts with existing code?

## Output
Record your verdict using ticket_review:
- --approve if all checks pass
- --reject --comment "<specific issues>" if any check fails

## Constraints
- Do NOT modify any code files
- Do NOT advance the ticket (human approves after you)
- Be specific about what's wrong — "file doesn't exist" not "needs work"
```

### 6.3 Agent orchestration

The dashboard's agent runner (Phase 7) orchestrates these. But agents can also be invoked manually:
```
claude --agent design-reviewer "Review ticket t-5a3f"
```

Or via the `/review` skill, which dispatches to the appropriate review agent(s) based on the ticket's current stage.

**Testable checkpoint:** Each agent can be invoked manually against a test ticket. Design-reviewer correctly validates file paths. Impl-reviewer checks acceptance criteria. Code-reviewer catches real issues.

---

## Phase 7: Dashboard + Orchestrator

**Repo:** `forge`
**Goal:** Update the web dashboard for pipeline view and build the stage-aware orchestrator.

### 7.1 Dashboard home: Unified Inbox + What's Next

The dashboard home page has two primary panels. Everything else is secondary navigation.

**Unified Inbox (left panel / top on mobile):**

The single view of "what needs me right now." Powered by `ticket_inbox` MCP tool / `GET /api/inbox`.

```
┌─ INBOX (7 items) ──────────────────────────────────────────────────────┐
│                                                                         │
│ ● REVIEW  api:t-5a3f  Auth service design      P1  Acme  · api repo   │
│   Design review needed · waiting 2h                                     │
│   [Approve] [Reject] [Open]                                            │
│                                                                         │
│ ● REVIEW  api:t-8b2c  Payment error handling   P0  Acme  · api repo   │
│   Code review needed · waiting 45m                                      │
│   [Approve] [Reject] [Open]                                            │
│                                                                         │
│ ● INPUT   webapp:t-9d1e  User onboarding flow  P2  Portal · webapp    │
│   Spec stage: build acceptance criteria                                 │
│   [Resume conversation] [Open]                                          │
│                                                                         │
│ ● INPUT   tk:t-3f7a  CLI help improvements     P3  (standalone) · tk  │
│   Triage stage: needs type and priority                                 │
│   [Resume conversation] [Open]                                          │
│                                                                         │
│ ● VERIFY  api:t-2c4b  Search performance fix   P1  Acme  · api repo   │
│   Verify stage: walk through acceptance criteria                        │
│   [Start verification] [Open]                                           │
│                                                                         │
│ ● DECIDE  webapp:t-6e8f  API rate limiting      P2  Portal · webapp   │
│   Agent rejected design — review needed                                 │
│   [View agent feedback] [Override] [Open]                               │
│                                                                         │
│ ● DECIDE  api:t-1a9c  Cache invalidation        P2  Acme  · api repo  │
│   Advisory gate: test coverage below 80%                                │
│   [Advance anyway] [Open]                                               │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

Inbox item types and their actions:
- **REVIEW**: Pending human review. Actions: Approve, Reject (inline with comment modal)
- **INPUT**: Conversational stage needing human. Action: Resume conversation (launches skill)
- **VERIFY**: Acceptance criteria walkthrough. Action: Start verification (launches /verify skill)
- **DECIDE**: Agent surfaced an issue that needs human judgment. Actions: view feedback, override, or send back

Sort order: Priority first, then waiting time (longest-waiting first within same priority). Critical (P0) items always surface at top regardless of age.

Filter controls: by project (epic), by repo, by action type, by assignee.

**What's Next (right panel / bottom on mobile):**

Per-project consolidated view of active work. Powered by `ticket_next` MCP tool / `GET /api/projects`.

```
┌─ ACTIVE PROJECTS ───────────────────────────────────────────────────┐
│                                                                      │
│ ▼ Acme project (api:t-0001)                  P1  ████████░░ 78%    │
│   12 tickets across api, webapp · 3 in inbox · 1 blocked            │
│   Next: Review auth service design (api:t-5a3f)                     │
│         Review payment error handling (api:t-8b2c)                   │
│         Unblock: api:t-7c3d waiting on api:t-5a3f                   │
│                                                                      │
│ ▼ Portal v2 (webapp:t-0042)                  P2  ████░░░░░░ 35%    │
│   8 tickets in webapp · 2 in inbox · 0 blocked                      │
│   Next: Build AC for onboarding flow (webapp:t-9d1e)                │
│         Decide on API rate limiting design (webapp:t-6e8f)           │
│                                                                      │
│ ▼ Standalone tickets                              ██████░░░░ 60%    │
│   4 tickets across tk, api · 1 in inbox · 0 blocked                 │
│   Next: Triage CLI help improvements (tk:t-3f7a)                    │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

Project summary shows:
- **Progress bar**: Fraction of total pipeline stages completed across all tickets in the project. A 7-stage feature at `test` = 5/7 complete. Weighted by ticket count.
- **Inbox count**: How many items in this project need you
- **Blocked count**: How many are stuck on dependencies
- **Next actions**: Top 3 highest-leverage actions for this project, derived from inbox items + ready tickets. "Finish what you started" — nearly-done tickets surface first.

Expanding a project shows the full pipeline view for that project's tickets.

**Pipeline view (secondary, per-project):**

When you expand/click a project, you see the pipeline board with ticket cards in stage columns.

### 7.1.1 Ticket Card (pipeline board)

The ticket card lives on the pipeline/kanban board. Each card must communicate its state in a single glance — what it is, where it's at, and what it needs.

**Design principles:**
1. Type is identity — color-coded left border, not a label you have to read
2. "What does this need right now" is the hero — not the title, not the metadata
3. Pipeline position is visible on every card (you shouldn't need the column header)
4. Review state is the primary interaction surface
5. Dense but not cluttered — whitespace separates zones of meaning

**Card anatomy (4 zones):**

```
┌─────────────────────────────────────────────┐
│ IDENTITY    What is this?                   │
│ ACTION      What does it need right now?    │
│ PROGRESS    Where is it in its pipeline?    │
│ META        Context at a glance             │
└─────────────────────────────────────────────┘
```

**Full card mockups by state:**

A feature waiting for design review:
```
┌──────────────────────────────────────────┐
│ ▓ feat  Auth service design        !! P1 │
│                                          │
│ ◉ Design review needed                   │
│   waiting 2h · requested by agent        │
│   [Approve] [Reject]                     │
│                                          │
│ ○───○───◉───○───○───○───○                │
│ tri  spc  des  imp  tst  vfy  done       │
│                                          │
│ api:t-5a3f · steve · 3d · 2 deps (1 ✓)  │
└──────────────────────────────────────────┘
```

A bug blocked on a dependency:
```
┌──────────────────────────────────────────┐
│ ▓ bug   Payment timeout error       ! P0 │
│                                          │
│ ⛔ Blocked on api:t-5a3f                  │
│   Auth service design (design stage)     │
│                                          │
│ ○───◉─ ─ ─○───○───○                      │
│ tri  imp      tst  vfy  done             │
│                                          │
│ api:t-7c3d · unassigned · 5d · 1 dep     │
└──────────────────────────────────────────┘
```

A task where an agent is actively working:
```
┌──────────────────────────────────────────┐
│ ▓ task  Refactor ID generation      P2   │
│                                          │
│ ⟳ Agent working: impl-agent              │
│   started 12m ago                        │
│                                          │
│ ○───○───●───○───○                        │
│ tri  imp  imp  tst  vfy  done            │
│         ↑ here                           │
│                                          │
│ tk:t-3f7a · claude · 1d · 0 deps         │
└──────────────────────────────────────────┘
```

A feature at a conversational stage needing human input:
```
┌──────────────────────────────────────────┐
│ ▓ feat  User onboarding flow        P2   │
│                                          │
│ 💬 Needs input: build acceptance criteria │
│   spec stage · last session 1d ago       │
│   [Resume conversation]                  │
│                                          │
│ ○───●───○───○───○───○───○                │
│ tri  spc  des  imp  tst  vfy  done       │
│                                          │
│ webapp:t-9d1e · steve · 4d · 0 deps      │
└──────────────────────────────────────────┘
```

A chore ready to advance (no blockers, gates pass):
```
┌──────────────────────────────────────────┐
│ ▓ chore Update CI config            P3   │
│                                          │
│ ✓ Ready to advance                       │
│   all gates pass                         │
│   [Advance to done]                      │
│                                          │
│ ○───●───○                                │
│ tri  imp  done                           │
│                                          │
│ tk:t-2c4b · steve · 2d · 0 deps          │
└──────────────────────────────────────────┘
```

**Zone details:**

**Identity zone (top row):**
- Left border color = type (feature=blue, bug=red, task=gray, epic=purple, chore=green)
- Type abbreviated: `feat`, `bug`, `task`, `epic`, `chore` — small, muted
- Title truncated to ~30 chars (full title on hover)
- Priority: `!!!` P0, `!!` P1, `!` P2, dim for P3-P4. Red text for P0.

**Action zone (middle, the hero):**
- Icon + bold primary action line:
  - `◉` Review needed (amber)
  - `⛔` Blocked (red)
  - `⟳` Agent working (blue, animated pulse)
  - `💬` Needs input (purple)
  - `✓` Ready to advance (green)
  - `✗` Review rejected (red)
- Secondary detail line: context about the action (who's it waiting on, how long, what's blocking)
- Inline action buttons when applicable (Approve/Reject for reviews, Advance for ready, Resume for input)

**Progress zone (pipeline mini-bar):**
- Horizontal dot track showing all stages for this ticket's type
- Filled dots = completed stages, current dot = highlighted, future = outline
- Dashed segments for skipped stages
- Stage abbreviations below for orientation (tri, spc, des, imp, tst, vfy, done)
- The mini-bar shows the ticket's *type-specific* pipeline (chores have 3 dots, features have 7)

**Meta zone (bottom row, muted):**
- Qualified ID (`api:t-5a3f`)
- Assignee (or `unassigned` in dim)
- Age since created (`3d`, `2w`)
- Dep summary: `2 deps (1 ✓)` — count with resolved fraction

**Interaction:**
- Click card → full detail slide-out panel (ticket body, review log, notes, conversations)
- Click action buttons → inline action (approve opens comment modal, advance triggers gate check)
- Drag card between columns → `ticket_advance` with confirmation if gates exist
- Right-click → context menu (assign, tag, link, skip stage)

**Visual states (card background/border):**
- Default: white/neutral
- Blocked: faint red tint on left border
- Agent working: faint blue pulse on border
- Review pending: amber left accent
- Ready: green left accent
- Stale (no activity > 3 days): subtle dimming, `stale` badge

### 7.1.2 Project Card (dashboard home)

The project card lives on the "What's Next" panel. It answers: "How is this project doing and what should I do about it?"

**Design principles:**
1. Progress is the hero — is this project moving or stuck?
2. Stage distribution tells the shape of work (bottleneck detection)
3. Next actions are the call to action — don't just show state, show what to do
4. Health signals surface problems without you having to drill in
5. Expandable: collapsed view is a summary, expanded shows full pipeline

**Card anatomy (5 zones):**

```
┌───────────────────────────────────────────────────────┐
│ HEADER       Title + priority + progress              │
│ DISTRIBUTION Where are tickets in the pipeline?       │
│ HEALTH       Key metrics at a glance                  │
│ ACTIONS      What should I do next?                   │
│ FOOTER       Repos, last activity                     │
└───────────────────────────────────────────────────────┘
```

**Full card mockups:**

Healthy project, mostly done:
```
┌───────────────────────────────────────────────────────┐
│                                                       │
│  Acme project                          !! P1          │
│  api:t-0001 · epic                                    │
│                                                       │
│  ████████████████████████████████░░░░░░░░  78%        │
│                                                       │
│  ┈┈┈┈┈┈╌╌╌╌╌╌╌╌╌┬───────┬───────┬──┬──┬─────────    │
│  triage  spec  design  impl   test vfy   done (8)     │
│                   1      2      1            ████      │
│                                                       │
│  12 active  ╷  3 need you  ╷  1 blocked  ╷  8 done   │
│                                                       │
│  → Review auth service design (api:t-5a3f) · 2h       │
│  → Review payment error handling (api:t-8b2c) · 45m   │
│  → Unblock api:t-7c3d (waiting on api:t-5a3f)         │
│                                                       │
│  api, webapp · last activity 45m ago                   │
│                                                       │
└───────────────────────────────────────────────────────┘
```

Project that's stalled:
```
┌───────────────────────────────────────────────────────┐
│                                                       │
│  Portal v2                             !  P2          │
│  webapp:t-0042 · epic                  ⚠ stalled      │
│                                                       │
│  ██████████████░░░░░░░░░░░░░░░░░░░░░░░░░░  35%       │
│                                                       │
│  ┬──────┬──────┈┈┈┈┈┈┈╌╌╌╌╌╌╌╌╌┈┈┈┈┈┈┈──────        │
│  triage  spec   design   impl   test  vfy  done (2)   │
│    1      3       2                          ██        │
│                                                       │
│   6 active  ╷  2 need you  ╷  0 blocked  ╷  2 done   │
│                                                       │
│  → Build AC for onboarding flow (webapp:t-9d1e)       │
│  → Decide on API rate limiting (webapp:t-6e8f)        │
│    ⚠ 3 tickets in spec with no activity for 2d        │
│                                                       │
│  webapp · last activity 2d ago                         │
│                                                       │
└───────────────────────────────────────────────────────┘
```

Small standalone group:
```
┌───────────────────────────────────────────────────────┐
│                                                       │
│  Standalone tickets                                   │
│  4 tickets across tk, api                             │
│                                                       │
│  ████████████████████████░░░░░░░░░░░░░░░░  60%        │
│                                                       │
│  ┬─────────────┬───────┬──────────────────            │
│  triage (1)     impl (1)   done (2)                   │
│                                                       │
│   2 active  ╷  1 needs you  ╷  0 blocked  ╷  2 done  │
│                                                       │
│  → Triage CLI help improvements (tk:t-3f7a)           │
│                                                       │
│  tk, api · last activity 6h ago                        │
│                                                       │
└───────────────────────────────────────────────────────┘
```

**Zone details:**

**Header zone:**
- Project title (epic title, or "Standalone tickets")
- Qualified ID + type badge (only for epics)
- Priority pips (same as ticket card)
- Health badge when something's wrong: `⚠ stalled` (no activity > 2d), `⛔ blocked` (all remaining tickets blocked), `✓ on track`

**Distribution zone (stage sparkline):**
- Horizontal segmented bar where segment width = proportion of tickets at each stage
- Labels below with counts for stages that have tickets
- Empty stages collapsed (no width) — you only see where tickets actually are
- "Done" shows as a solid filled block at the end with count
- Visual bottleneck detection: if one stage is disproportionately wide, it's a pileup

**Health zone (metrics row):**
- `N active` — non-done, non-triage tickets
- `N need you` — inbox items for this project (amber if > 0)
- `N blocked` — tickets with unresolved deps (red if > 0)
- `N done` — completed tickets
- Inline dividers for visual separation

**Actions zone (next steps):**
- Top 3 next actions from `ProjectSummary.NextActions`, ordered by priority then pipeline position
- Each action shows: arrow → action description (ticket ID) · waiting time
- Stale warning if applicable: `⚠ N tickets at stage with no activity for Xd`
- Actions are clickable — "Review..." opens review modal, "Build AC..." resumes conversation

**Footer zone (muted):**
- Repos involved (comma-separated badges)
- Last activity timestamp (relative: "45m ago", "2d ago")
- Stale projects get amber last-activity text

**Interaction:**
- Click card → expand to show full pipeline board for this project (the column view with ticket cards)
- Click a next-action → opens the relevant ticket card or triggers the action directly
- Hover health badge → tooltip with details ("3 tickets with no activity: t-9d1e, t-6e8f, t-4b2a")

**Visual states (card treatment):**
- On track: clean white card, green progress bar
- Needs attention: amber left border, amber "need you" count
- Stalled: full amber border, `⚠ stalled` badge, dimmed progress bar
- Blocked: red left border when all remaining tickets are blocked
- Complete (100%): green progress bar, celebration state (subtle), auto-collapses after 1 day

### 7.1.3 Design system notes

**Color language (consistent across both cards):**
- **Blue**: feature type, agent activity
- **Red**: bug type, blocked state, P0 priority
- **Green**: chore type, ready/advance/done states
- **Purple**: epic type, conversational/human-input stages
- **Amber/Yellow**: review pending, needs attention, stale warnings
- **Gray**: task type, muted metadata, inactive states

**Typography hierarchy:**
- Title: medium weight, 14px, primary text color
- Action line: bold, 13px, action-color (matches state)
- Meta/footer: regular, 11px, muted text color
- Priority pips: bold, action-colored (`!` = amber, `!!` = orange, `!!!` = red)

**Density controls:**
The dashboard should offer density settings:
- **Comfortable**: full cards as shown above (default)
- **Compact**: collapse progress zone and meta zone into single line:
  ```
  ┌──────────────────────────────────────────┐
  │ ▓ feat  Auth service design        !! P1 │
  │ ◉ Design review needed · 2h  [Approve]  │
  │ ○─○─◉─○─○─○─○ · api:t-5a3f · steve · 3d │
  └──────────────────────────────────────────┘
  ```
- **List**: single row per ticket (for very large ticket sets):
  ```
  ▓ api:t-5a3f  feat  Auth service design  P1  ◉ review  des  steve  3d
  ```

**Stage actions (from any view):**
- "Advance" button: triggers `ticket_advance` via API, shows gate check results
- "Review" buttons: approve/reject with comment modal
- "Run Agent" button: dispatches appropriate agent for current stage
- "Auto-advance" toggle: run full pipeline autonomously up to a specified stop-stage

### 7.2 Stage-aware orchestrator

The orchestrator replaces the simple agent runner. It's an API endpoint that:

1. Accepts: `POST /orchestrate { ticketId, targetStage?, auto? }`
2. Reads ticket state via `ticket_show`
3. Determines what to do next based on current stage
4. Dispatches appropriate agent(s)
5. Checks gate results
6. If auto mode + gates pass → advances and continues to next stage
7. If gate fails → stops, reports, waits for human intervention
8. Streams progress via SSE (existing infrastructure)

Orchestration flow example (feature ticket at `design`, targetStage=`test`, auto=true):
```
1. Check gate: design → implement (design exists? review approved?)
   → Gate passes
2. Advance to implement
3. Dispatch: implement agent
   → Agent codes, commits
4. Dispatch: impl-review agent
   → APPROVED
5. Dispatch: code-review agent
   → APPROVED
6. Check gate: implement → test (mandatory: both reviews approved)
   → Gate passes
7. Advance to test
8. Dispatch: test-runner agent
   → Tests pass
9. Stop: targetStage reached
10. Notify human: "t-5a3f ready for verification"
```

### 7.3 API endpoints

New/modified endpoints:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/inbox` | GET | Unified inbox: items needing human attention |
| `/api/projects` | GET | Active projects with progress + next actions |
| `/api/dashboard` | GET | Combined inbox + projects + stats (single call for page load) |
| `/api/tickets` | GET | List with stage/review filters |
| `/api/tickets/:id` | GET | Full ticket detail |
| `/api/tickets/:id/advance` | POST | Advance ticket (with gate checks) |
| `/api/tickets/:id/review` | POST | Record review verdict |
| `/api/tickets/:id/skip` | POST | Skip stages |
| `/api/tickets/:id/log` | GET | Stage transition history |
| `/api/pipeline` | GET | All tickets grouped by stage |
| `/api/pipeline/:epicId` | GET | Pipeline view for a specific project |
| `/api/orchestrate` | POST | Start orchestrated pipeline run |
| `/api/orchestrate/:runId` | GET | Check orchestration status |
| `/api/orchestrate/:runId/events` | GET (SSE) | Stream orchestration progress |

The dashboard server starts `tk serve --workspace ...` and communicates via MCP. All reads go through `ticket_inbox`, `ticket_next`, `ticket_dashboard`. All writes go through `ticket_advance`, `ticket_review`, etc. The workspace MCP server handles routing writes to the correct repo.

### 7.4 Learning integration

- Session summaries in `data/sessions/` auto-link to tickets via `conversations` field
- Pattern detection creates tickets with `tag: pattern-action`
- Dashboard home shows: velocity, active patterns, recent learnings
- Nightly pipeline writes discovered learnings into relevant ticket Notes sections

**Testable checkpoint:** Dashboard home loads with inbox and project panels. Inbox shows correct items (reviews, conversational stages, agent escalations) sorted by priority then age. What's Next shows per-project progress and top actions. Clicking a project expands to pipeline view. Inline approve/reject works from inbox. "Resume conversation" launches correct skill. Auto-advance runs a ticket through stages with SSE progress updates.

---

## Phase 8: Integration Testing + Polish

**Repo:** both `tk` and `forge`
**Goal:** End-to-end testing of the full pipeline, from `/triage` to `done`.

### 8.1 End-to-end test scenario

Walk a feature ticket through the full pipeline:

```bash
# 1. Triage (conversational — manual for now)
# Use /triage skill, creates ticket at triage stage

# 2. Advance to spec
tk advance t-xxx   # gate: description exists ✓

# 3. Spec agent runs (or /spec skill)
# Builds acceptance criteria, scope, context

# 4. Human approves spec
tk review t-xxx --approve --actor "human:steve"

# 5. Advance to design
tk advance t-xxx   # gate: AC exists, review=approved ✓

# 6. Design agent runs
# Writes design, implementation plan

# 7. Design review agent runs (advisory)
# Validates against codebase

# 8. Human approves design
tk review t-xxx --approve --actor "human:steve"

# 9. Advance to implement
tk advance t-xxx   # gate: design+plan exist, review=approved ✓

# 10. Implement agent runs
# Writes code

# 11. Impl review agent runs (mandatory)
tk review t-xxx --approve --actor "agent:impl-reviewer"

# 12. Code review agent runs (mandatory)
tk review t-xxx --approve --actor "agent:code-reviewer"

# 13. Advance to test
tk advance t-xxx   # gate: both reviews approved ✓ (MANDATORY)

# 14. Test agent runs
# Records test results

# 15. Advance to verify
tk advance t-xxx   # gate: tests pass ✓

# 16. Human verifies
tk review t-xxx --approve --actor "human:steve"

# 17. Advance to done
tk advance t-xxx   # gate: review=approved ✓
```

### 8.2 Test the type-dependent pipelines

- Bug: triage → implement → test → verify → done (skip spec, design)
- Chore: triage → implement → done (skip spec, design, test, verify)
- Epic: triage → spec → design → done (no implementation)

### 8.3 Test edge cases

- Skip: bug that needs design → `tk skip t-xxx --to design --reason "complex bug"`
- Rejection: design review rejects → ticket stays at design, agent revises
- Mandatory gate failure: impl review rejects → can't advance past implement
- Advisory gate: design review rejects but human overrides → advance anyway, logged
- Migration: old tickets with `status` field → `tk migrate` converts correctly
- Propagation: all children reach `done` → parent auto-advances

### 8.4 MCP integration test

- Skill invokes `ticket_create` via MCP → ticket created at triage
- Skill invokes `ticket_advance` → ticket moves to next stage
- Agent invokes `ticket_review` → review recorded
- Dashboard calls API → API calls MCP → state changes reflected

---

## Dependency Graph

```
Phase 1 (pipeline core)
   │
   ▼
Phase 2 (CLI + MCP)
   │
   ├──────────────────────┐
   ▼                      ▼
Phase 3 (TUI + release)  Phase 4 (forge repo setup)
                            │
                            ├──────────────────┐
                            ▼                  ▼
                          Phase 5 (skills)    Phase 6 (agents)
                            │                  │
                            └────────┬─────────┘
                                     ▼
                              Phase 7 (dashboard + orchestrator)
                                     │
                                     ▼
                              Phase 8 (integration testing)
```

Phases 3 and 4 can run in parallel after Phase 2.
Phases 5 and 6 can run in parallel after Phase 4.
Phase 7 depends on 5 and 6.
Phase 8 is the final integration pass.

---

## Effort Estimates (Rough)

| Phase | Scope | Parallelizable with |
|-------|-------|-------------------|
| **Phase 1** | ~15 files in `pkg/ticket/`, pure Go library work | Nothing (foundation) |
| **Phase 2** | ~10 CLI files + ~300 lines MCP, test updates | Nothing (depends on 1) |
| **Phase 3** | TUI + release mechanics | Phase 4 |
| **Phase 4** | Repo setup, file moves, path updates | Phase 3 |
| **Phase 5** | ~10 skill files, MCP-first rewrite | Phase 6 |
| **Phase 6** | ~5 agent definitions | Phase 5 |
| **Phase 7** | Dashboard frontend + orchestrator backend | Nothing (depends on 5+6) |
| **Phase 8** | Integration tests, polish | Nothing (depends on 7) |

---

## Risk Mitigation

1. **Migration breaks existing tickets:** `tk migrate --dry-run` shows changes without writing. Dual support for one release. Test against your real `.tickets/` directory.

2. **MCP server becomes bottleneck:** The MCP server is just a thin wrapper over `pkg/ticket/`. If MCP is slow, the CLI is always available as fallback. They share the same core.

3. **Skills too rigid with MCP-only:** Skills can still include bash blocks for non-ticket operations (git, build, test). Only ticket operations move to MCP.

4. **Dashboard rewrite too large:** Phase 7 can be split further — pipeline view first, then orchestrator, then learning integration. The dashboard can work with just the API (CLI/MCP backend) before the orchestrator exists.

5. **Agent quality varies:** Start with advisory-only agents (Phase 6), then flip the `implement → test` gate to mandatory after you've validated the agents produce useful reviews.
