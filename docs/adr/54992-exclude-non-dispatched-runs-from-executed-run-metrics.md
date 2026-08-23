# ADR-54992: Exclude Non-Dispatched Runs from Executed-Run Metrics

**Date**: 2026-08-23
**Status**: Draft
**Deciders**: Unknown

---

### Context

GitHub Actions creates workflow run records before evaluating any workflow-level conditions. For command workflows triggered by `issue_comment` events (such as `q`), this means:

- **`skipped` runs** are created when the activation condition (`if:`) evaluates to false — for example, a comment that does not contain the workflow's command keyword. No job is ever dispatched.
- **`action_required` runs** are created when a bot actor's comment triggers the workflow but the repository requires manual approval for bot-actor workflow runs. GitHub holds the run indefinitely; no job is dispatched.

Prior to this change, the run-collection code filtered `skipped` and `cancelled` runs but kept `action_required` runs. Because the `q` workflow is triggered by every `Copilot` bot comment in the repository, `action_required` runs accumulated at high volume, inflating the total-run denominator and collapsing the reported success rate to ~0.8% (2/261 runs) even though every dispatched run succeeded. The `gh aw health`, `gh aw logs`, and forecast subsystems all consumed the same unfiltered run set.

### Decision

We will introduce a single shared predicate `isNonDispatchedConclusion(conclusion string) bool` that returns `true` for `"skipped"` and `"action_required"`, and apply it consistently across all three run-collection and metric-computation paths: `listWorkflowRunsWithPagination` (health/logs), `isCompletedDispatchedRun` (forecast training and validation), and any future paths that consume workflow run sets. Runs matching this predicate are excluded before they reach any metric aggregation, so success-rate, duration, and token metrics reflect only runs that actually dispatched an agentic job.

Because the exclusion happens after GitHub has applied the API batch limit, health pagination advances its cursor from the oldest *raw* run in each batch and terminates on the raw batch size rather than on the filtered result. Otherwise a batch consisting entirely of non-dispatched runs would filter down to zero and be misread as "no more runs", hiding older dispatched runs.

### Alternatives Considered

#### Alternative 1: Filter at the reporting/display layer only

Keep `action_required` runs in the in-memory run set and suppress them only when computing and rendering summary metrics. This avoids touching the core run-collection path.

**Why not chosen**: The run set is consumed by multiple subsystems (health, logs, forecast). Filtering at the display layer would require each subsystem to independently re-implement the exclusion logic, risking inconsistency and future regressions. Centralizing the predicate at collection time is simpler and more robust.

#### Alternative 2: Suppress phantom runs at the source via repository settings

Change the repository's bot-actor workflow approval setting so that `Copilot`-actor comments do not create `action_required` runs in the first place.

**Why not chosen**: Changing the repository approval setting affects all workflows and all bot actors, not just `q`. It requires a repository-level administrative change outside the scope of this codebase, and is explicitly noted as out-of-scope in the PR. The metrics fix is necessary regardless because historical `action_required` runs already exist.

### Consequences

#### Positive
- Command workflows with heavy bot-actor activity (e.g., `q`) now report accurate success rates, removing a false signal that was causing unnecessary investigation.
- The exclusion logic is centralized in one predicate (`isNonDispatchedConclusion`), making future extension (e.g., adding another never-dispatched conclusion type) a one-line change with a single test update.

#### Negative
- If GitHub changes the semantics of `action_required` (e.g., a run that is manually approved later and does dispatch a job is still marked `action_required` in the conclusion field), the predicate would incorrectly exclude those runs. This would require a revisit of the filtering strategy.
- The fix treats the symptom (filter phantom runs) rather than the root cause (GitHub creating phantom runs for bot-actor comments). Bot-actor comment events will continue to accumulate `action_required` runs in the GitHub UI.

#### Neutral
- Health run collection now issues additional API batches when early batches are dominated by non-dispatched runs, up to the existing `MaxIterations` cap, so the reported sample size stays stable instead of collapsing to zero.
- `cancelled` runs are filtered separately (in `listWorkflowRunsWithPagination`) and are not included in `isNonDispatchedConclusion` because cancellation can happen after a job dispatches, making it distinct from never-dispatched runs. This distinction is preserved.
- The new test file `logs_non_dispatched_runs_test.go` documents the expected exclusion behavior and the reporting impact, providing a regression guard for future changes to run filtering.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
