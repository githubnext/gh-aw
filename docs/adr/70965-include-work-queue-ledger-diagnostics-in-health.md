# ADR-70965: Include Work-Queue Ledger Diagnostics in `gh aw health`

**Date**: 2026-10-21
**Status**: Proposed
**Deciders**: Repository maintainers through PR #70965 review

---

### Context

`gh aw health` reports workflow health using GitHub Actions run outcomes: success rates, trends, durations, and failure inventories. Workflows that use the work queue can fail in ways that never appear in run outcomes. Examples include a growing backlog, pools that cannot schedule work, paused admission or grants, and failed deliveries. Some of these workflows may have no recent runs at all, so a run-based report can show them as quiet or healthy when they are actually stalled. The work-queue ledger (the `work-queue` branch) already holds this state, and `pkg/workqueue` can already read, replay, and summarize it (`workQueueStatsAt`). Health must stay read-only, stay within its shared API budget, avoid downloading artifacts, and avoid modifying the user's checkout.

### Decision

Extend `gh aw health` to detect work-queue usage and report current ledger diagnostics **after** collecting primary Actions evidence. A workflow counts as using the work queue when its **published default-branch lock file** sets `GH_AW_WORK_QUEUE_ENABLED: "true"` at the workflow, job, or step `env` level. This deliberately uses compiler-emitted environment entries, not examples or JSON embedded in scripts. Detection shares the remaining health API budget and reuses committed Git objects and downloaded lock content; exhausting that remaining budget must not displace already collected Actions evidence.

When at least one selected workflow uses the work queue, health reads the repository's queue ledger **once** through the existing authenticated, isolated `workqueue.Branch` Git transport. The selected GitHub host applies both to transport and to resource-reference validation during read and replay. Cross-host resource references remain invalid. Health checks that a policy is installed and that the ledger's repository matches the health target, then computes metadata-only diagnostics at the current time.

The result appears as `collection.work_queue` in JSON and as `[health_work_queue]` in text. Status is `attention_required` only when admission or grants are paused or any delivery failed, `unavailable` when ledger evidence cannot be read, and `observed` otherwise. Detection warnings remain separate from ledger status: an unreadable unrelated workflow cannot degrade an otherwise observed ledger. Public warnings use payload-free categories and recovery guidance instead of raw API responses or transport diagnostics. `observed` describes a ledger snapshot, not live worker health. Queue diagnostics do not depend on `--days` or the date window, and do not alter Actions success rates or fleet counts.

### Alternatives Considered

#### Alternative 1: Keep work-queue diagnostics in the dedicated work-queue CLI only

Users could keep using the existing work-queue stats commands while health stays based only on Actions runs. This needs no new code in health. It was rejected because health is the main fleet-status command, and the PR's stated problem is that success rates hide queue backlog, blockers, pauses, and failed deliveries. If a user has to run a second command, stalled workflows with no recent runs stay invisible in health.

#### Alternative 2: Infer work-queue health from Actions runs or artifacts

Health could infer queue state from worker or dispatcher run outcomes, or from downloaded run artifacts. This was rejected for three reasons. Health intentionally avoids downloading artifacts. Run outcomes cannot show scheduling blockers, pauses, or delivery verification state. And workflows with no runs in the window would produce no evidence at all. The ledger is the authoritative source of queue state.

#### Alternative 3: Detect work-queue usage from local workflow sources or the checkout's current branch

Detection could parse local `.md` sources or lock files in the working tree. This alternative is rejected in this decision because health targets the published default-branch configuration, possibly in another repository (`--repo`), and the local checkout may be stale, dirty, on another branch, or a different repository altogether. Committed local Git objects may accelerate reading the published revision, but working-tree contents are not authoritative.

#### Alternative 4: Inspect queue configuration before collecting Actions evidence

This ordering could reuse detection reads during subsequent run analysis. It is rejected because repositories with many workflows can exhaust the shared API budget before the failure inventory or run samples are collected. Queue evidence is optional enrichment and must use only the budget left after primary collection.

### Consequences

#### Positive
- Health surfaces backlog, pool blockers, oldest available work, pauses, native reservations, and delivery state, including for workflows with no recent runs.
- Missing or invalid definitions, policy, ledger, credentials, or a repository mismatch appear as explicit warnings instead of looking healthy.
- `workqueue.Branch` gains an explicit `Host`, so ledger reads (and REST clients) work against non-default GitHub hosts.
- The JSON change is additive (`work_queue` field plus a new `work_queue` capability).
- Public warnings omit raw response bodies, stored payloads, and credential diagnostics while retaining actionable access and timeout guidance.

#### Negative
- Health makes extra API calls (repository metadata plus one lock-file blob per selected workflow) and performs a Git ledger read with a 30-second timeout, which makes it slower and adds new failure modes.
- Detection depends on the compiler emitting at least one plain `GH_AW_WORK_QUEUE_ENABLED` environment marker. Compiler contract regression coverage must preserve this invariant; JSON text in a script alone is not a detection signal.
- Queue detection may be incomplete when primary Actions collection consumes the shared budget; explicit warnings are preferable to sacrificing the failure-first report.
- The report mixes two kinds of evidence, windowed run samples and a current ledger snapshot, which users have to tell apart.
- An `observed` status can be misread as "workers are healthy", even though it only means the ledger snapshot could be read.

#### Neutral
- Queue diagnostics are computed once per repository even when several selected workflows share the queue.
- Text rendering reuses existing work-queue formatters (`workQueueStatsText`, `workQueuePoolDiagnosticsText`), so health output and work-queue CLI output stay the same.
- Actions success rates, fleet workflow counts, and `healthy_workflows` membership do not change.

---

This record is complete for review. Acceptance remains part of the maintainers' pull request decision.
