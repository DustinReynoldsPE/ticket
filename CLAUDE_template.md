## Task Management

This project uses `tk`, a CLI ticket system for persistent task management. Tickets live in `.tickets/` as markdown files with YAML frontmatter.

### Quick Reference

```bash
tk help                    # Full command reference
tk ready                   # What's ready to work on
tk show <id>               # View ticket details
tk create "title" -t task  # Create a ticket
tk advance <id>            # Move to next pipeline stage
```

### Pipeline & Stages

Tickets move through a type-dependent pipeline:

```
triage → [spec → design →] implement → [test → verify →] done
```

- **feature**: triage → spec → design → implement → test → verify → done
- **bug/task**: triage → implement → test → verify → done
- **chore**: triage → implement → done
- **epic**: triage → spec → design → done

### Gates

Stage transitions enforce gate checks (e.g., "description exists", "review approved"). When a gate blocks you:

1. Read the error — it tells you exactly what's missing
2. Fix it (e.g., `tk edit <id> -d "description"`, `tk review <id> --approve`)
3. Retry `tk advance <id>`

### Completing Tickets

`done` is the terminal stage — there is no separate "closed" status. To reach done:

- **Normal path**: satisfy all gates, then `tk advance <id>`
- **Skip path**: `tk skip <id> --to done --reason "justification"` (bypasses gates with audit trail)
- **`--force` cannot reach done** — use `tk skip` instead

### Common Patterns

```bash
# Work on next ready ticket
tk ready
tk advance <id>                              # triage → implement (if description exists)

# Record a review (required by some gates)
tk review <id> --approve

# Skip stages you don't need
tk skip <id> --to implement --reason "trivial fix, no spec needed"

# Complete a ticket directly
tk skip <id> --to done --reason "verified manually, tests pass"

# Add context to a ticket
tk edit <id> -d "description of the work"
tk add-note <id> "found the root cause in auth.go"
```

### What NOT to Do

- Do not use `--status` — there is no status field, only `--stage`
- Do not use `tk advance --force` to reach done — it will fail; use `tk skip` instead
- Do not guess commands — run `tk help` if unsure
