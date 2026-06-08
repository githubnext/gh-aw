---
name: pr-finisher
description: Prepare an open pull request for merge from a GitHub Copilot cloud agent. Drives Reviews, local validation, and Mergeable to a ready state. Does not merge, and cannot trigger CI.
---

# PR Finisher

Drive an open PR for the current branch to a merge-ready state from a **GitHub Copilot cloud agent**. **Do not merge.** When everything you can act on is done, report ready-for-human-merge and stop.

## Execution context

This skill runs inside a GitHub Copilot cloud agent, not on a developer's machine.

- **The agent's pushes do not trigger CI.** Workflows on the PR will not re-run after the agent commits. Any existing `statusCheckRollup` reflects an earlier HEAD and goes stale the moment the agent pushes.
- **Local `make` targets are the agent's authoritative correctness signal** before push. CI is observational only.
- **Re-running CI is a hand-off to a human** (close/reopen the PR, `workflow_dispatch`, or a push from a maintainer). The agent must surface this in its summary.
- **No watch / no sleep loops.** The agent has no async wait state; one pass + summary + stop.

## Three merge-ready conditions

A PR is merge-ready when **all three** are satisfied. Work them **concurrently**.

| Condition | Definition | Agent's signal |
|---|---|---|
| **Reviews** | Every unresolved review thread is addressed on its merits, replied to, and resolved. Code changes alone do not satisfy this. | `copilot-review` skill + GraphQL `reviewThreads` |
| **Checks** | Local `make fmt` / `make lint` / `make test-unit` / `make test` pass. Last-known CI runs reviewed at log level. | `make` targets locally; `gh pr checks` / `gh run view --log-failed` for prior runs |
| **Mergeable** | PR is OPEN, not draft, `mergeable: MERGEABLE`, not `BEHIND` if the repo requires up-to-date branches. | `gh pr view --json mergeable,mergeStateStatus,state,isDraft` |

Because the agent cannot re-trigger CI, "Checks" is satisfied at the agent's level when **local validation passes and prior CI failures have been root-caused and fixed in the pushed commits**. Final green CI requires a human to re-trigger after the agent stops.

Top-level PR comments and review bodies are useful feedback but **not** a merge gate. Read and action useful ones; do not block on them.

## Hard rules

- **Do not merge.** Never run `gh pr merge`, enable auto-merge, or enqueue. This skill stops at "ready for merge."
- **Do not post stand-alone PR comments.** Only reply on existing review threads / comments that need a response. Do not ping reviewers or CODEOWNERS.
- **Always disable pagers** for `gh`: prefix with `GH_PAGER=""` or pipe through `cat`. Without this, commands hang in non-interactive shells.
- **Never wait for CI to re-run.** No `bash sleep`, no `gh run watch`, no `gh pr checks --watch`, no re-check loop after push. The agent's pushes will not trigger workflows; waiting is futile.
- **Local validation is non-negotiable before push.** Because CI will not re-run, the only correctness gate the agent gets is `make ...` locally. Treat a green local run as the bar.
- **Reviews are not done until reply + resolve both succeed.** Code change alone ≠ thread handled.
- **Smallest fix that works.** Don't change unrelated code. Fix lint before tests.
- **Pre-existing unrelated failures** → identify explicitly in the summary; do not guess-fix.

## CI-fix anti-patterns (do not do these)

A failing CI step is a signal, not a nuisance. Even though the agent cannot re-run CI to confirm, the following are **forbidden** and should trigger `ask_user` instead:

- Disabling, skipping, or neutering shared tooling (build caches, lint rules, type checks, env vars, required checks) to make a failure go away.
- "Temporary" disables with a TODO to re-enable later. They outlive the PR and become permanent.
- Lowering coverage thresholds, removing assertions, or loosening a test until it passes. If the test is wrong about product behavior, fix its **logic** (assertions, fixtures, setup); don't relax it.
- Bundling a workaround with a real fix ("belt and suspenders"). Ship one real fix or escalate. Never both.
- Special-casing one OS/runner to hide a failure on that platform.

**Anti-pattern test:** if the change would make the failure invisible on future PRs without solving it, stop and escalate.

**Before declaring a tool broken on a platform:** reproduce locally, check version/config, look for transient causes (timeouts, network, runner state). Most "X is broken on macOS/Windows" reports are transient flakes on healthy tooling.

**For flaky infra** (caches, registries, runners): prefer narrow fixes — targeted retry, higher timeout, pre-flight health check. If a narrow fix doesn't land in one or two attempts, escalate via `ask_user`.

## Workflow

The agent runs this once. There is no monitoring loop.

### 1. Triage

```bash
GH_PAGER="" gh pr view <number> --json state,isDraft,reviewDecision,mergeable,mergeStateStatus,statusCheckRollup,headRefOid
GH_PAGER="" gh pr checks <number>
```

If merged/closed, report and stop. Otherwise classify each condition as ✅ / ❌ / ⏳ / ❓. The CI snapshot here is your **only** view of CI for this run — capture which checks failed and why before changing anything, because after you push it will be stale.

### 2. Address Reviews

Delegate to the `copilot-review` skill. For each unresolved thread: make change → commit → reply → resolve. A thread is not handled until reply + resolve both succeed. Push only after all in-scope threads are handled (batch with other fixes below).

### 3. Address Mergeable

```bash
GH_PAGER="" gh pr view <number> --json mergeable,mergeStateStatus
```

- `CONFLICTING` → resolve conflicts using the repo's conventions. If you cannot determine the correct resolution, `ask_user`.
- `mergeStateStatus: BEHIND` → update branch from base. After updating, scan the new commits for tooling drift (lockfiles, toolchains, lint configs); re-run installs if manifests changed, and flag drift in the summary so any new errors read as drift, not regressions.

### 4. Address Checks (local + prior CI)

**Local validation** — the agent's only correctness signal. Run in order; fix at each step before moving on:

```bash
make fmt
make lint
make test-unit
make test
```

If a `make test` fix changes wasm compiler output, or wasm golden tests fail:

```bash
make update-wasm-golden
```

Then re-run the affected tests.

**Prior CI failures** — for each failure captured during triage, pull logs and fix the root cause:

```bash
GH_PAGER="" gh run view <run_id> --log-failed
```

Classify as: real product/test bug, infra flake, or third-party flake. Apply the fix in the agent's commits and, where possible, **reproduce the fix locally** via the matching `make` target. If the failure can't be reproduced locally (infra-only), state that in the summary so the human re-triggers CI with eyes open. Per anti-pattern rules: 1–2 narrow attempts, then `ask_user`.

### 5. Push and stop

Push once, after all fixes are in. **Do not re-check `gh pr checks` expecting a new run.** Print the summary and stop.

## Summary format

At the stopping point, print:

```
- ✅ Reviews — <plain language>
- ✅ Checks (local) — <plain language>
- <status> Checks (CI) — stale after agent push; needs human re-trigger. Prior failures: <fixed | open | not reproducible locally>
- ✅ Mergeable — <plain language>

Actions taken: <what changed in this run>
Hand-off: CI must be re-triggered by a maintainer (close/reopen PR, workflow_dispatch, or push) before merge.
Still needed: <human review, anything not actionable from the agent>
```

Status vocabulary:
- ✅ satisfied — checked and passing
- ❌ failing — checked and failing
- ⏳ pending — running, waiting for signal (rare for the agent; never use for the post-push CI state)
- ❓ unknown — could not be checked (API error, indeterminate, or CI stale after agent push). Never use ❌ for this.

**Translate status into plain language.** Don't write bare labels. Always state explicitly that CI on the agent's HEAD is unverified until a human re-triggers it.

## Stopping conditions

- **Ready for merge (pending human CI re-trigger)** — local validation green, Reviews resolved, Mergeable clean. Summarize and stop.
- **Nothing actionable remains** — non-actionable blocker (human approval, external service). Summarize and stop.
- **Truly stuck** — unresolvable conflicts, ambiguous feedback, irreproducible failures. `ask_user` with context.

## Completion standard

The task is complete only when all are true:

- `make fmt`, `make lint`, `make test-unit` all pass (or unrelated pre-existing failures explicitly identified).
- `make test` was run and fixed when it was part of the failing state; wasm goldens regenerated when required.
- The `copilot-review` skill addressed all in-scope review threads (reply + resolve succeeded for each).
- Mergeable condition was checked; conflicts resolved and `BEHIND` updated when present.
- Prior CI failures were inspected at the log level and either fixed at the root cause (with a local reproduction where possible) or explicitly flagged as not locally reproducible / escalated.
- Final changes were pushed exactly once at the end. No post-push re-check loop.
- A structured ✅/❌/⏳/❓ summary was printed, including an explicit hand-off line for the human CI re-trigger.
- No `gh pr merge` was run.
