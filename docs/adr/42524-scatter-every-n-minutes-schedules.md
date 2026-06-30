# ADR-42524: Scatter Every-N-Minutes Schedules via Fuzzy Token Pipeline

**Date**: 2026-06-30
**Status**: Draft
**Deciders**: Unknown

---

### Context

Workflows using "every N minutes" natural-language schedule syntax (e.g., `every 10 minutes`, `every 5m`) were compiled directly to standard cron `*/N * * * *` expressions. Because all workflows sharing the same interval produce the same cron pattern, they all fire simultaneously, creating predictable load spikes at minute boundaries. The codebase already contained a fuzzy-scatter pipeline (`ScatterSchedule`) that resolves intermediate `FUZZY:*` tokens to deterministically offset cron expressions for hourly and weekly schedules. Minute-interval schedules were not included in this pipeline, leaving a gap in the thundering-herd mitigation strategy.

### Decision

We will extend the existing fuzzy-scatter pipeline to cover minute-interval schedules. The schedule parser now emits `FUZZY:EVERY_MINUTE/N * * * *` instead of `*/N * * * *` for "every N minutes" inputs. A new `handleEveryMinute` handler in the scatter layer resolves this token to `M/N * * * *`, where `M = stableHash(workflowIdentifier, N) ∈ [0, N-1]`, preserving the period while distributing start minutes across the clock face. The offset is deterministic per workflow identifier so recompilation produces the same cron without drift. Raw cron expressions (`*/N * * * *` typed directly by users) bypass scatter as before.

### Alternatives Considered

#### Alternative 1: Keep `*/N * * * *` and accept simultaneous firing

The simplest option: do nothing. Workflows with the same interval continue to fire at the same minute. This was rejected because simultaneous firing creates measurable load spikes, and the infrastructure to mitigate it already exists for other schedule types — extending it to minute intervals is low-cost and consistent.

#### Alternative 2: Apply a random offset at compile time

Assign a random offset `M ∈ [0, N-1]` when compiling the schedule, storing the result as a plain cron expression. This avoids the intermediate `FUZZY:` token. It was rejected because a random offset changes on every recompilation, causing schedule drift — a workflow would silently shift its firing time whenever its definition is reprocessed. The deterministic hash approach used in the decision ensures the same workflow always produces the same cron, which is safer for audit trails and change detection.

#### Alternative 3: Offset by a fixed per-repo value rather than per-workflow hash

Use a single fixed offset derived from the repository rather than the workflow identifier. This was rejected because all workflows in the same repo would still cluster together at the same offset, merely shifting the load spike rather than distributing it.

### Consequences

#### Positive
- Workflows sharing a minute interval are now spread across the clock face, reducing simultaneous firing and the associated load spikes.
- The offset is deterministic — same workflow identifier always produces the same cron, so recompilation is idempotent and auditable.
- The fix reuses the existing `stableHash` and `ScatterSchedule` infrastructure, keeping the implementation surface small and consistent with the hourly/weekly scatter approach.

#### Negative
- The compiled cron output changes from the intuitive `*/N` form to the less-familiar `M/N` step-with-start form. Both are semantically equivalent but the latter may surprise users reading raw cron values.
- Existing stored cron expressions (`*/N * * * *`) generated before this change are not retroactively updated; workflows will only receive a scattered offset on their next recompilation.

#### Neutral
- The intermediate `FUZZY:EVERY_MINUTE/N` token must now be understood by any tooling that inspects or stores cron expressions mid-pipeline.
- Tests for minute-interval schedules now assert offset-in-range (`[0, N-1]`) rather than exact cron strings, which is more robust to hash-function changes but loses exact value pinning (except in the cross-platform stability test).

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
