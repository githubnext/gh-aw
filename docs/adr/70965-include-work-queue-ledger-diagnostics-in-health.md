# ADR-70965: Include Work-Queue Ledger Diagnostics in `gh aw health`

**Date**: 2026-10-21
**Status**: Draft
**Deciders**: pelikhan, [TODO: verify reviewers]

---

### Context

`gh aw health` reports workflow health using GitHub Actions run outcomes: success rates, trends, durations, and failure inventories. Workflows that use the work queue can fail in ways that never appear in run outcomes. Examples include a growing backlog, pools that cannot schedule work, paused admission or grants, and failed deliveries. Some of these workflows may have no recent runs at all, so a run-based report can show them as quiet or healthy when they are actually stalled. The work-queue ledger (the `work-queue` branch) already holds this state, and `pkg/workqueue` can already read, replay, and summarize it (`workQueueStatsAt`). Health must stay read-only, stay within its shared API budget, avoid downloading artifacts, and avoid modifying the user's checkout.

### Decision

We will extend `gh aw health` to detect work-queue usage and report current ledger diagnostics. A workflow counts as using the work queue when its **published default-branch lock file** sets `GH_AW_WORK_QUEUE_ENABLED: "true"` at the workflow, job, or step `env` level. Detection reuses the health API budget and the committed Git object cache. When at least one selected workflow uses the work queue, health reads the repository's queue ledger **once**. It uses the existing authenticated, isolated `workqueue.Branch` Git transport, with a new `Host` field so the selected GitHub host is honored. It then replays the ledger to the current time and checks that a policy is installed and that the ledger's repository matches the health target. The result appears as `collection.work_queue` in JSON and as `[health_work_queue]` in text. Status is `attention_required` when admission or grants are paused or any delivery failed, `unavailable` (with explicit warnings) when evidence cannot be read, and `observed` otherwise. `observed` describes a ledger snapshot and is not a claim that workers are healthy. Queue diagnostics do not depend on the `--days` or date window, and existing Actions success rates and fleet counts stay unchanged.

### Alternatives Considered

#### Alternative 1: Keep work-queue diagnostics in the dedicated work-queue CLI only

Users could keep using the existing work-queue stats commands while health stays based only on Actions runs. This needs no new code in health. It was rejected because health is the main fleet-status command, and the PR's stated problem is that success rates hide queue backlog, blockers, pauses, and failed deliveries. If a user has to run a second command, stalled workflows with no recent runs stay invisible in health.

#### Alternative 2: Infer work-queue health from Actions runs or artifacts

Health could infer queue state from worker or dispatcher run outcomes, or from downloaded run artifacts. This was rejected for three reasons. Health intentionally avoids downloading artifacts. Run outcomes cannot show scheduling blockers, pauses, or delivery verification state. And workflows with no runs in the window would produce no evidence at all. The ledger is the authoritative source of queue state.

#### Alternative 3: Detect work-queue usage from local workflow sources or the checkout's current branch

Detection could parse local `.md` sources or lock files in the working tree. This was rejected because health targets the published default-branch configuration, possibly in another repository (`--repo`), and the local checkout may be stale, on another branch, or a different repository altogether. [TODO: verify whether this alternative was explicitly considered]

### Consequences

#### Positive
- Health surfaces backlog, pool blockers, oldest available work, pauses, native reservations, and delivery state, including for workflows with no recent runs.
- Missing or invalid definitions, policy, ledger, credentials, or a repository mismatch appear as explicit warnings instead of looking healthy.
- `workqueue.Branch` gains an explicit `Host`, so ledger reads (and REST clients) work against non-default GitHub hosts.
- The JSON schema change is additive (`work_queue` field plus a new `work_queue` capability), so existing consumers keep working.

#### Negative
- Health makes extra API calls (repository metadata plus one lock-file blob per selected workflow) and performs a Git ledger read with a 30-second timeout, which makes it slower and adds new failure modes.
- Detection depends on the compiled `GH_AW_WORK_QUEUE_ENABLED` env marker in lock files. If the compiler changes how it emits that marker, detection silently breaks.
- The report mixes two kinds of evidence, windowed run samples and a current ledger snapshot, which users have to tell apart.
- An `observed` status can be misread as "workers are healthy", even though it only means the ledger snapshot could be read.

#### Neutral
- Queue diagnostics are computed once per repository even when several selected workflows share the queue.
- Text rendering reuses existing work-queue formatters (`workQueueStatsText`, `workQueuePoolDiagnosticsText`), so health output and work-queue CLI output stay the same.
- Actions success rates, fleet workflow counts, and `healthy_workflows` membership do not change.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
