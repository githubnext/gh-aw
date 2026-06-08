---
name: pr-finisher
description: Prepare an open pull request for merge by satisfying the Reviews, Checks, and Mergeable conditions for the current branch. Does not merge.
---

# PR Finisher

Drive an open PR for the current branch to a merge-ready state. **Do not merge.** When all conditions are satisfied, report ready-for-human-merge and stop.

## Three merge-ready conditions

A PR is merge-ready when **all three** are satisfied simultaneously. Work them **concurrently**, not sequentially.

| Condition | Definition | Primary signal |
|---|---|---|
| **Reviews** | Every unresolved review thread is addressed on its merits, replied to, and resolved. Code changes alone do not satisfy this. | `copilot-review` skill + GraphQL `reviewThreads` |
| **Checks** | All required CI checks pass on the current HEAD, and local `make fmt`/`make lint`/`make test-unit`/`make test` pass. | `gh pr checks` + `make` targets |
| **Mergeable** | PR is OPEN, not draft, `mergeable: MERGEABLE`, not `BEHIND` if the repo requires up-to-date branches. | `gh pr view --json mergeable,mergeStateStatus,state,isDraft` |

Top-level PR comments and review bodies are useful feedback but are **not** a merge gate (GitHub has no resolve state for them). Read and action useful ones; do not block on them.

## Hard rules

- **Do not merge.** Never run `gh pr merge`, enable auto-merge, or enqueue. This skill stops at "ready for merge."
- **Do not post stand-alone PR comments.** Only reply on existing review threads / comments that need a response. Do not ping reviewers or CODEOWNERS.
- **Always disable pagers** for `gh`: prefix with `GH_PAGER=""` or pipe through `cat`. Without this, commands hang in non-interactive shells.
- **Never block waiting for CI.** No `bash sleep`, no `gh run watch`, no `gh pr checks --watch`. After pushing, do **one** immediate re-check, then end the turn if checks are still pending.
- **Reviews are not done until reply + resolve both succeed.** Code change alone ≠ thread handled.
- **Smallest fix that works.** Don't change unrelated code. Fix lint before tests.
- **Pre-existing unrelated failures** → identify explicitly in the summary; do not guess-fix.

## CI-fix anti-patterns (do not do these)

A failing CI step is a signal, not a nuisance. The following are **forbidden** and should trigger `ask_user` instead:

- Disabling, skipping, or neutering shared tooling (build caches, lint rules, type checks, env vars, required checks) to make a failure go away.
- "Temporary" disables with a TODO to re-enable later — they outlive the PR and become permanent.
- Lowering coverage thresholds, removing assertions, or loosening a test until it passes. If the test is wrong about product behavior, fix its **logic** (assertions, fixtures, setup) — don't relax it.
- Bundling a workaround with a real fix ("belt and suspenders"). Ship one real fix or escalate. Never both.
- Special-casing one OS/runner to hide a failure on that platform.

**Anti-pattern test:** if the change would make the failure invisible on future PRs without solving it, stop and escalate.

**Before declaring a tool broken on a platform:** reproduce, check version/config, look for transient causes (timeouts, network, runner state). Most "X is broken on macOS/Windows" reports are transient flakes on healthy tooling.

**For flaky infra** (caches, registries, runners): prefer narrow fixes — targeted retry, higher timeout, pre-flight health check. If a narrow fix doesn't land in one or two attempts, escalate via `ask_user`.

## Workflow

### 1. Triage

```bash
GH_PAGER="" gh pr view <number> --json state,isDraft,reviewDecision,mergeable,mergeStateStatus,statusCheckRollup,headRefOid
GH_PAGER="" gh pr checks <number>
```

If merged/closed, report and stop. Otherwise classify each condition as ✅ satisfied / ❌ failing / ⏳ pending / ❓ unknown. This is your starting checklist.

### 2. First pass — push every fix you can see now

Address everything visible before entering any wait loop so CI runs against the final state.

**Reviews** — delegate to the `copilot-review` skill to address all in-scope review threads. For each thread, the workflow is: make change → commit → push → reply → resolve. A thread is not handled until reply + resolve both succeed.

**Checks (local)** — run in order; fix at each step before moving on:

```bash
make fmt
make lint
make test-unit
make test            # only after lint + test-unit are clean, or if CI shows broader test failure
```

If a `make test` fix changes wasm compiler output, or wasm golden tests fail:

```bash
make update-wasm-golden
```

Then re-run the affected tests.

**Checks (CI)** — for each failing check, fetch logs and find the root cause:

```bash
GH_PAGER="" gh run view <run_id> --log-failed
```

Classify as: real product/test bug, infra flake, or third-party flake. Fix at the source per the anti-pattern rules above. If a narrow fix doesn't work in 1–2 attempts, escalate via `ask_user` — do not broaden the workaround.

**Mergeable** — check and act:

```bash
GH_PAGER="" gh pr view <number> --json mergeable,mergeStateStatus
```

- `CONFLICTING` → resolve conflicts using the repo's conventions (rebase or merge). If you cannot determine the correct resolution, `ask_user`.
- `mergeStateStatus: BEHIND` → update branch from base so CI runs against the final state. After updating, scan the new commits for tooling drift (lockfiles, toolchains, lint configs); re-run installs if manifests changed and flag drift so new errors read as drift, not regressions.

### 3. Verify and converge

After pushing fixes, do **one** immediate re-check to confirm CI picked up the new HEAD:

```bash
GH_PAGER="" gh pr checks <number>
GH_PAGER="" gh pr view <number> --json mergeable,mergeStateStatus,reviewDecision
```

If checks are still running and Reviews + Mergeable are satisfied, **end the turn**. Do not sleep, do not poll. The user (or next tick) will re-invoke when there's new signal.

If a new failure appears, return to step 2 for that condition only.

### 4. Stop

Stop when one of:

- **Ready for merge** — all three conditions satisfied. Report and stop. Do not merge.
- **Only Checks pending** — Reviews + Mergeable satisfied, CI still running. Summarize and end the turn.
- **Nothing actionable remains** — a non-actionable blocker (e.g., human approval required, persistent external failure). Summarize and stop.
- **Truly stuck** — unresolvable conflicts, ambiguous feedback, repeated CI failures. Use `ask_user` with context.

## Summary format

At every stopping point, print a structured summary:

```
- ✅ Reviews — <plain language>
- ✅ Checks — <plain language>
- ✅ Mergeable — <plain language>

Actions taken: <what changed since last check-in>
Still needed: <what blocks merge, if anything>
```

Status vocabulary:
- ✅ satisfied — checked and passing
- ❌ failing — checked and failing
- ⏳ pending — running, waiting for signal
- ❓ unknown — could not be checked (API error, indeterminate); never use ❌ for this case

**Translate status into plain language.** Don't write bare labels like "Checks pending." Write what is actually true, e.g. "Required checks passed; Cloud Code Review was not requested for this head, so merge is not waiting on it."

## Completion standard

The task is complete only when all are true:

- `make fmt`, `make lint`, `make test-unit` all pass (or unrelated pre-existing failures explicitly identified).
- `make test` was run and fixed when it was part of the failing state; wasm goldens regenerated when required.
- The `copilot-review` skill addressed all in-scope review threads (reply + resolve succeeded for each).
- Mergeable condition was checked; conflicts resolved and `BEHIND` updated when present.
- Failing CI checks were inspected at the log level and either fixed at the root cause or escalated.
- Final changes were pushed and one re-check confirmed CI picked up the new HEAD.
- A structured ✅/❌/⏳/❓ summary was printed.
- No `gh pr merge` was run.
