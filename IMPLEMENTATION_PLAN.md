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

    // Parsed from markdown, not stored in frontmatter.
    Title string `yaml:"-"`
    Body  string `yaml:"-"`
    Notes []Note `yaml:"-"`
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

### 1.10 Tests

- `pipeline_test.go`: Pipeline definitions, NextStage, HasStage for all types
- `gates_test.go`: Gate checks for every transition, mandatory vs advisory
- `workflow_test.go`: Update existing + add Advance, Skip, SetReview tests
- `migrate_test.go`: All status→stage mappings, idempotency, round-trip
- `format_test.go`: Update for new fields, backward compat with old format

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

### 2.2 New MCP tools

| Tool | Description |
|------|-------------|
| `ticket_advance` | Advance a ticket through its pipeline |
| `ticket_review` | Record a review verdict |
| `ticket_skip` | Skip to a non-adjacent stage |
| `ticket_log` | Get stage transition history |
| `ticket_pipeline` | Get tickets grouped by pipeline stage |
| `ticket_migrate` | Run migration (status→stage) |

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

**Testable checkpoint:** `tk advance`, `tk review`, `tk pipeline` work from CLI. MCP tools work via `tk serve`. `tk migrate --dry-run` shows what would change. Full `test-suite.sh` passes. Old commands still work with deprecation warnings.

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

### 7.1 Dashboard updates

**Pipeline view (replaces kanban):**
- Columns are pipeline stages, not statuses
- Type selector filters to show type-specific pipeline
- Tickets show: title, priority, review state, assignee
- Color coding: review=pending (yellow), review=approved (green), review=rejected (red)
- Click ticket to expand: full details, review log, conversations, notes

**Stage actions:**
- "Advance" button: triggers `ticket_advance` via API, shows gate check results
- "Review" buttons: approve/reject with comment modal
- "Run Agent" button: dispatches appropriate agent for current stage
- "Auto-advance" toggle: run full pipeline autonomously up to a specified stop-stage

**Review panel:**
- Shows pending reviews across all tickets
- Filter by review type (design, impl, code)
- Quick approve/reject from the panel

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
| `/api/tickets` | GET | List with stage/review filters |
| `/api/tickets/:id/advance` | POST | Advance ticket (with gate checks) |
| `/api/tickets/:id/review` | POST | Record review verdict |
| `/api/tickets/:id/skip` | POST | Skip stages |
| `/api/tickets/:id/log` | GET | Stage transition history |
| `/api/pipeline` | GET | All tickets grouped by stage |
| `/api/orchestrate` | POST | Start orchestrated pipeline run |
| `/api/orchestrate/:runId` | GET | Check orchestration status |
| `/api/orchestrate/:runId/events` | GET (SSE) | Stream orchestration progress |

All ticket-mutating endpoints delegate to the `tk` MCP server (or call `pkg/ticket` directly if forge is Go-based, or shell to `tk` CLI as fallback).

### 7.4 Learning integration

- Session summaries in `data/sessions/` auto-link to tickets via `conversations` field
- Pattern detection creates tickets with `tag: pattern-action`
- Dashboard home shows: velocity, active patterns, recent learnings
- Nightly pipeline writes discovered learnings into relevant ticket Notes sections

**Testable checkpoint:** Dashboard shows pipeline view. Clicking "Advance" triggers gate checks and advances. "Run Agent" dispatches the right agent. Auto-advance runs a ticket through multiple stages. SSE updates show real-time progress.

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
