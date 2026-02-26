# Competitive Review: Agentic Workflow vs. Best-in-Class (2026)

**Date:** 2025-02-25
**Scope:** Rating our agentic workflow redesign against the best-in-class systems shipping today.

---

## Overall: Architecturally ahead of most, behind on execution and autonomy

The redesign is more thoughtfully designed than ~90% of what exists in the wild. But the best-in-class systems have shipped what we've documented. Here's the breakdown by dimension.

---

## 1. Workflow Structure & Enforcement

**Our system: 9/10**
**Best-in-class comparator: Kiro (AWS)**

Our 7-stage pipeline with type-dependent paths and structural gate enforcement is genuinely best-in-class *as a design*. The insight that workflow should live in the ticket system, not in prompt text, is something most teams haven't figured out yet.

[Kiro](https://kiro.dev/blog/kiro-and-the-future-of-software-development/) has the closest analogy with its spec-driven development (requirements → design → tasks), including EARS notation for acceptance criteria — which we independently adopted. But Kiro's pipeline is 3 phases and fixed. Ours is 7 stages, type-dependent, with explicit skip-with-audit-trail. That's more mature.

**Where we lead:** Type-dependent pipelines (feature vs bug vs chore vs epic getting different stage sequences) is something nobody else does structurally. Kiro treats everything the same. Devin and Cursor have no concept of pipeline at all.

**Where we're behind:** Kiro's agent hooks (auto-trigger on file save/create/delete) are shipping today. Our orchestrator is still on paper.

---

## 2. Agent Architecture (Evaluator-Optimizer)

**Our system: 9/10**
**Best-in-class comparator: Anthropic's reference patterns, OpenHands**

Our decomposition into focused specialist agents (spec-builder, design-reviewer, impl-reviewer, code-reviewer, test-runner) directly implements [Anthropic's evaluator-optimizer pattern](https://www.anthropic.com/research/building-effective-agents) and orchestrator-workers pattern. The key insight — "agents don't control flow, `tk` controls flow" — is exactly what Anthropic recommends: keep agents simple, let infrastructure handle orchestration.

[OpenHands](https://openhands.dev/) achieves 72% on SWE-Bench Verified with their modular SDK, but their agents are general-purpose, not role-specialized. [MetaGPT](https://github.com/geekan/MetaGPT) simulates a full product team (PM, tech lead, developer, analyst) as separate agents — conceptually similar to ours, but theirs is a framework for others to use, while ours is a bespoke system tuned to our workflow.

**Where we lead:** The mandatory vs. advisory review distinction (mandatory at implement→test, advisory elsewhere) is nuanced in a way that frameworks like CrewAI and LangGraph don't express. Most systems are all-or-nothing on gates.

**Where we're behind:** No parallel agent execution. [Cursor 2.0](https://www.builder.io/blog/devin-vs-cursor) runs up to 8 agents in parallel on the same project. [Factory AI](https://factory.ai) parallelizes "Droids" at massive scale for migrations and maintenance. Our orchestrator is sequential.

---

## 3. Human-in-the-Loop Design

**Our system: 10/10**
**Best-in-class comparator: None — we're at the frontier**

This is our strongest dimension. The explicit per-stage default modes (conversational vs. autonomous), the `--auto` flag for low-risk work, the risk tiering (low/normal/high/critical) with risk-scaled gates — this is more sophisticated than anything shipping today.

[Devin](https://www.builder.io/blog/devin-vs-cursor) is fully async: delegate and review later. Cursor is fully synchronous: pair-program in real-time. Factory AI sits in between. None of them let you *configure* the human involvement level per stage and per risk level the way our design does.

The "hands-free mode" (`tk advance --auto --through verify`) for P3-P4 chores, with mandatory human verify at the end, is exactly the right trust model. [Anthropic's 2026 Agentic Coding Trends Report](https://resources.anthropic.com/2026-agentic-coding-trends-report) describes the shift from "assistance" to "augmentation" to "autonomy" — our system lets you dial that per-ticket, which is ahead of the curve.

**Where we lead:** Risk-aware gate scaling. A "critical" security ticket gets human review at every stage; a P4 chore runs fully autonomous until verify. Nobody else has this granularity.

---

## 4. State Management & Observability

**Our system: 8/10**
**Best-in-class comparator: Factory AI, Linear + AI integrations**

Our filesystem-as-database approach (markdown + YAML frontmatter in git) is elegant and differentiated. The review log, conversation links, and stage transition history give full traceability. `tk pipeline` and `tk log` make the state queryable.

**Where we lead:** Git-native state means every transition is versioned, diffable, and mergeable. No database migrations, no vendor lock-in. The unified inbox (`tk inbox`) computing "what needs me" from ticket state is a clean derived-state approach.

**Where we're behind:** No real-time streaming of agent progress. Factory AI and Devin show live agent activity. Our manager dashboard polls at 500ms, which works but isn't SSE/WebSocket for agent message streaming. Also, modern tools like [Linear](https://linear.app) have rich analytics dashboards (cycle time, throughput, bottleneck detection) that our pipeline health metrics (gate rejection rate, stage dwell time) aspire to but haven't built yet.

---

## 5. Learning & Knowledge Feedback

**Our system: 8/10**
**Best-in-class comparator: Nobody — this is a unique capability**

The learning extraction pipeline (PreCompact → SessionEnd → nightly extraction → pattern detection → rollups) is genuinely novel. No mainstream tool does this. Devin, Cursor, Copilot, Factory — none of them have a concept of "learning from past sessions and feeding patterns back into the workflow."

The redesign's improvement (ticket-linked learnings, pattern → actionable ticket pipeline, 3+ occurrences auto-creates chore tickets) is smart.

**Where we lead:** Institutional memory. The idea that a coding pattern detected 3 times auto-suggests a CLAUDE.md addition, or a failure pattern auto-suggests a lint rule, is genuine competitive advantage.

**Where we're behind:** It's batch (nightly), not real-time. The best-in-class vision would be: agent finishes a task, learns something, that learning is immediately available to the next agent run. Our pipeline has a 24-hour feedback delay.

---

## 6. Ecosystem Integration

**Our system: 4/10**
**Best-in-class comparator: Factory AI, Cursor, GitHub Copilot**

This is our weakest dimension. Our system is a closed, bespoke stack for a solo developer (or very small team).

[Factory AI](https://factory.ai) integrates with IDE, CLI, Slack, Linear, CI/CD, and triggers agents from issue assignment. [GitHub Copilot](https://github.com/features/copilot) is embedded in the world's largest developer platform. [Cursor](https://cursor.com) inherits the entire VS Code extension ecosystem.

Our system requires: a custom Go CLI, a custom Bun web server, custom Claude Code skills, a specific directory layout, and tmux. That's a lot of moving parts with no third-party integrations.

**Where we lead:** MCP-first design means the ticket system is accessible to any MCP-compatible tool, which is forward-looking.

**Where we're behind:** No CI/CD integration, no PR automation, no Slack/Discord notifications, no integration with existing project management tools. The `forge` consolidation helps but doesn't solve this.

---

## 7. Scalability & Multi-Project

**Our system: 7/10**
**Best-in-class comparator: Factory AI, Devin**

The workspace mode for multi-repo ticket aggregation and the unified inbox are good designs. But they're single-user constructs.

Factory AI runs hundreds of parallel agents across an organization's repos. Devin handles concurrent sessions across teams. Our system is fundamentally designed for one person orchestrating agents across their personal projects.

**Where we lead:** The multi-repo workspace with cross-project ticket views is well-designed for the solo/small-team use case.

**Where we're behind:** No multi-user support, no team permissions, no concurrent agent execution across projects.

---

## Scorecard Summary

| Dimension | Our Score | Best-in-Class | Gap |
|-----------|-----------|--------------|-----|
| Workflow structure & enforcement | 9/10 | Kiro (7/10) | **We lead** |
| Agent architecture | 9/10 | OpenHands (9/10) | Tied on design, they lead on execution |
| Human-in-the-loop | 10/10 | Devin/Cursor (6/10) | **We lead significantly** |
| State management & observability | 8/10 | Factory/Linear (9/10) | Slight gap |
| Learning & knowledge feedback | 8/10 | Nobody (0/10) | **Unique capability** |
| Ecosystem integration | 4/10 | Factory/GitHub (9/10) | Major gap |
| Scalability & multi-project | 7/10 | Factory (9/10) | Moderate gap |

**Weighted overall: ~8/10 on design, ~5/10 on shipped reality**

---

## The Honest Take

Our system's *design* is best-in-class for a bespoke, single-developer agentic workflow. The structural gate enforcement, type-dependent pipelines, risk-aware human-in-the-loop, and learning feedback loop are ahead of what's commercially available. We've independently arrived at patterns that Anthropic recommends and that Kiro partially implements.

### Three things holding it back from unqualified best-in-class:

1. **It's not shipped yet.** The redesign branch has zero code changes. Kiro, Factory, and Cursor are shipping these patterns to users today. Design quality matters less than execution velocity in this space — the landscape is moving fast.

2. **No parallelism.** The sequential agent orchestration is a generation behind Cursor's 8-parallel-agents and Factory's fleet-scale execution. Adding git-worktree-based parallel agent runs would close this gap.

3. **Closed ecosystem.** The bespoke stack means we're the only user, which means we're the only person finding bugs, requesting features, and validating the design. The learning feedback loop partially compensates, but there's no substitute for real-world usage pressure.

### Highest-leverage next move:

Ship Phase 1 (stage pipeline in `tk`). Everything else in the design depends on it, and it's the piece that's most differentiated from what exists commercially.

---

## Sources

- [Anthropic - Building Effective Agents](https://www.anthropic.com/research/building-effective-agents)
- [Anthropic - 2026 Agentic Coding Trends Report](https://resources.anthropic.com/2026-agentic-coding-trends-report)
- [Kiro - Spec-Driven Development](https://kiro.dev/blog/kiro-and-the-future-of-software-development/)
- [Martin Fowler - Understanding Spec-Driven Development: Kiro, spec-kit, and Tessl](https://martinfowler.com/articles/exploring-gen-ai/sdd-3-tools.html)
- [Factory AI](https://factory.ai)
- [OpenHands](https://openhands.dev/)
- [Devin vs Cursor Comparison](https://www.builder.io/blog/devin-vs-cursor)
- [Faros AI - Best AI Coding Agents 2026](https://www.faros.ai/blog/best-ai-coding-agents-2026)
