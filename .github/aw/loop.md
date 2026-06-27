---
description: Deep research synthesis of loop-engineering workflow patterns from githubnext/autoloop, githubnext/goal, and githubnext/crane, with implementation guidance for gh-aw workflows.
---

# Loop Engineering Patterns

This document distills loop-engineering patterns from:

- https://raw.githubusercontent.com/githubnext/autoloop/main/workflows/autoloop.md
- https://raw.githubusercontent.com/githubnext/goal/main/workflows/goal.md
- https://raw.githubusercontent.com/githubnext/crane/main/workflows/crane.md

These are upstream sources on `main`; re-check upstream workflow files when adopting this guidance in case patterns evolve.

Use it as a playbook when building long-running, iterative, agentic workflows.

## What “loop engineering” means

A loop workflow repeatedly:

1. selects exactly one work item,
2. makes one bounded improvement step,
3. verifies with concrete evidence,
4. preserves accepted progress,
5. records durable state for the next run.

The loop must stay reliable across many runs, branch merges, CI failures, and human steering.

## Shared architecture across Autoloop, Goal, and Crane

### 1) Single-item scheduler

All three workflows select one item per run from many candidates:

- Autoloop: one program
- Goal: one goal issue
- Crane: one migration

This keeps run cost bounded and creates fair round-robin progress.

### 2) Canonical long-running branch + single PR

Each item owns one stable branch and one draft PR:

- `autoloop/<program>`
- `goal/<issue>-<slug>`
- `crane/<migration>`

The loop accumulates accepted commits on that same PR over time.

### 3) Ratcheting acceptance

A change is accepted only when it improves the tracked metric (or advances the contract) and passes CI/verification gates.

If improvement fails, the change is discarded and the run is still recorded.

### 4) Durable state in repo-memory

All state is persisted as markdown in a dedicated memory branch:

- `memory/autoloop`
- `memory/goal`
- `memory/crane`

State is both machine-readable and human-editable.

### 5) Human control-plane issue

Each item has one canonical issue with:

- a durable status comment sentinel (`<!-- ...:STATUS -->`),
- one per-run log comment,
- human steering directives.

### 6) Explicit no-progress and pause semantics

When the system is blocked or stuck, it pauses loudly with a concrete reason instead of silently retrying forever.

## Pattern inventory

### Pattern A — Item selection and fairness

Use a deterministic pre-step scheduler that writes a compact selection artifact (for example `/tmp/gh-aw/autoloop.json` in gh-aw runners, with the file name matching your workflow name) with:

- selected item,
- deferred items,
- due/not-due flags,
- existing PR/branch metadata.

Do not let the agent discover candidates ad hoc in-prompt.

### Pattern B — Canonical branch invariants

Branch naming must be deterministic and suffix-free.

Always use ahead/behind logic against default branch:

- `ahead=0, behind>0`: fast-forward/reset branch to default,
- `ahead>0, behind>0`: merge default into branch,
- else: checkout as-is.

When a force-push is required by your branch-sync strategy, use `--force-with-lease` (not `--force`).
`--force-with-lease` still needs caution, so keep canonical branches single-writer (the workflow) to minimize concurrent push conflicts.

### Pattern C — One PR per item

Never create multiple active PRs for the same item.

Resolution order:

1. scheduler-provided `existing_pr`,
2. state-file PR fallback,
3. create exactly one PR if none exists.

### Pattern D — Improve → push → gate → accept

Adopt a three-phase accept path:

1. metric/contract improvement check,
2. push and wait for CI/checks,
3. accept only on green.

This avoids sandbox-only false positives.

### Pattern E — CI fix loop with circuit breakers

When CI fails after an improved change:

- collect failing jobs and error signatures,
- attempt bounded fix retries,
- stop on repeated identical signature,
- pause with structured reason (`ci-fix-exhausted`, `stuck`, `ci-timeout`).

### Pattern F — Structured state file

Keep a stable state layout with:

- machine-state table (iteration count, last run, best metric, pause/completion fields),
- current focus/checkpoint,
- lessons learned,
- foreclosed avenues/blockers,
- iteration history (newest first).

This creates continuity across runs and enables deterministic scheduling.

### Pattern G — Setup guard and safety rails

Use sentinel-based configuration checks before first real run:

- Autoloop: `<!-- AUTOLOOP:UNCONFIGURED -->`
- Crane: `<!-- CRANE:UNCONFIGURED -->`

If unconfigured, create/refresh a setup issue and skip execution.

### Pattern H — Direction-aware metrics

Support both optimization directions:

- `higher` is better (default),
- `lower` is better.

Use direction in:

- improvement test,
- signed delta reporting,
- target-metric completion check.

### Pattern I — Completion by evidence, not intent

Completion requires explicit evidence gates.

- Goal enforces issue-defined completion contracts.
- Crane separates reaching target metric from deterministic completion-gate pass.
- Autoloop supports target-metric completion with explicit label transition.

Never mark complete on belief.

### Pattern J — Unified run reporting

On every run (accepted/rejected/error/blocked):

- update durable status comment,
- append per-run summary comment,
- include run URL, checkpoint, evidence, result, next step.

This creates an auditable narrative.

## Comparative notes by project

| Project | Primary loop unit | Unique strength | Key reusable pattern |
|---|---|---|---|
| Autoloop | Program | General metric-driven optimization with rich iteration memory | Improvement ratchet + CI-gated accept/reject |
| Goal | Goal-labeled issue | Contract-first execution and definition-quality gating | “Needs action” path before implementation |
| Crane | Migration | Milestone plan + strategy selection (`in-place` vs `greenfield`) | iteration 0 planning commit and migration-specific completion gate |

## Implementation blueprint for new loop workflows in gh-aw

When creating a new loop workflow, implement in this order:

1. **Define the loop unit** (issue/program/migration/task).
2. **Add scheduler pre-step** that selects one item and emits JSON context.
3. **Define canonical branch and single-PR invariant**.
4. **Add durable state schema** in repo-memory.
5. **Implement run phases**: read state → choose checkpoint → change → verify.
6. **Add accept/reject logic** with direction-aware metric handling.
7. **Gate acceptance on CI/check health**.
8. **Add bounded fix loop** with failure-signature no-progress guard.
9. **Implement completion semantics** with explicit evidence gate.
10. **Add status + per-run issue comments** for observability.
11. **Add pause/recovery policy** for blocked or repeated failures.
12. **Document command-mode overrides** (slash command steering).

## Minimal loop run-state model

Use these conceptual states:

- `active`
- `accepted`
- `rejected`
- `error`
- `needs_action`
- `blocked`
- `paused`
- `completed`

Transitions must be deterministic and evidence-backed.

## Common failure modes to avoid

- Branch name drift (suffixes/hashes/run IDs)
- Multiple PRs per item
- Marking completion without deterministic evidence
- Repeating the same failed CI signature without pause
- Losing long-term context by storing state only in ephemeral run logs
- Unbounded scope growth per iteration

## Practical guidance for prompt authors

For loop prompts, explicitly require:

- one checkpoint per run,
- smallest useful change,
- explicit evidence command output,
- explicit `noop`/blocked behavior,
- state updates every run,
- strict branch/PR invariants.

Keep prompts short; move durable policy to state + structured workflow rules.

## Reusable checklist

- [ ] One selected item per run
- [ ] Canonical branch name with no suffix
- [ ] Single draft PR per item
- [ ] Durable state file updated every run
- [ ] Improvement criterion defined (direction-aware)
- [ ] Acceptance gated on CI/checks
- [ ] Fix-loop retry cap and signature-based stop
- [ ] Explicit blocked/paused handling
- [ ] Deterministic completion gate
- [ ] Status comment + per-run comment updated
