# ADR-71218: Move Outcome Evaluation into the `logs` Command

**Date**: 2026-10-22
**Status**: Draft
**Deciders**: pelikhan (PR author); maintainer acceptance pending review of [#71218](https://github.com/github/gh-aw/pull/71218)

---

### Context

Live outcome evaluation checks the current GitHub state of safe outputs produced by workflow runs. A separate `gh aw outcomes` command exposed this evaluation independently of the run selection and artifact collection provided by `gh aw logs`. Users need outcome observations for the same runs selected through workflow discovery, multiple targets, stdin, or episode retrieval, without a second selection and collection command. The requested CLI direction is to bring `--outcomes` to `logs` and remove `gh aw outcomes`, while preserving historical scoring, offline episode joins, and JSONL export.

### Decision

Make live outcome evaluation opt-in through `gh aw logs --outcomes`, including `logs episode --outcomes`, and remove the standalone `gh aw outcomes` command rather than retain an alias. Composability is the primary driver: evaluate the runs already selected and collected by `logs`, reuse their safe-output manifests, and include the `outcomes` artifact set when needed. Expose per-run reports in `.outcomes[]` and selected-episode rollups in `.episode_outcomes` of the logs JSON document; move offline helpers to `gh aw logs outcomes-history` and `gh aw logs outcomes-episodes`, with JSONL observations exported through `--outcomes-dir` or `GH_AW_OUTCOMES_DIR`.

Keep collection lightweight: the `outcomes` artifact set downloads the compact executed safe-output manifest, not full agent logs. Episode retrieval collects outcome manifests only for verified episode members, after validating identity and run-attempt evidence; already collected manifests are reused. Missing manifests produce explicit `eval_error` coverage diagnostics rather than an apparently successful empty result.

Combine `--episodes --outcomes` to include recorded episode membership and hops alongside the in-memory outcome join. The same join powers `logs outcomes-episodes` for separately persisted reports; the combined flags do not launch that command, require JSONL export, make additional join-specific GitHub requests, or expand the selected membership.

Evaluate current state on each invocation, even when artifact data is cached. Reuse successful, validated live API responses across the selected runs within that invocation, with request identity including the GitHub host; coalesce identical in-flight reads and reuse labels already present on fetched issue or PR objects. Do not persist this live-response cache across invocations or retain failed responses as reusable observations. Paginated REST reads request 100 items per page, and the objective impact report batches selected runs into one CLI invocation so that this reuse applies across the report.

### Alternatives Considered

#### Alternative 1: Keep the standalone `gh aw outcomes` command

A separate command keeps the existing CLI stable and gives live evaluation its own surface. It does not satisfy the requested removal or make outcomes part of the same selection and JSON document as logs. Keeping parallel public collection paths would also require maintaining their selection and evidence handling.

#### Alternative 2: Add `logs --outcomes` and retain a deprecated alias

An alias or wrapper would ease migration for existing scripts. It is not chosen because the requested interface is a single live-evaluation surface with the old command removed. The major changeset and migration instructions make this an explicit breaking change; no deprecation period is part of this decision.

#### Alternative 3: Pipe logs into a separate outcome evaluator

An evaluator consuming `logs --json` or a collected directory could share run selection while remaining independently composable. It would still require a second command and a separate output contract instead of attaching observations and episode rollups directly to the logs report. Integration keeps consumers such as the objective impact report on one collection interface.

### Consequences

#### Positive

- Outcome evaluation composes with existing logs selection modes and returns run data and outcome observations together.
- Compact manifests and invocation-scoped API reuse reduce repeated downloads and live reads without turning cached artifacts into stale outcome observations.
- The impact report consumes CLI classifications in one batch instead of duplicating live evaluation.

#### Negative

- Removing `gh aw outcomes` breaks existing scripts and requires a major release. For one run, migrate to `printf '%s\n' <run-id> | gh aw logs --stdin --repo owner/repo --outcomes --json`.
- Opt-in evaluation adds live API requests and latency. Episode outcome checks are additional to the episode retrieval bounds; request reuse does not make evaluation free or guarantee a transactionally consistent snapshot.
- Logs gains outcome-specific flags, offline subcommands, and JSON fields that must remain consistent across its collection and rendering paths.

#### Neutral

- Historical scoring and offline episode joins keep their separate roles under new `logs` command paths.
- JSONL export remains available for downstream processing. Missing evidence and individual evaluation failures remain explicit rather than being interpreted as rejection or zero coverage.
- This record documents the implementation for review; it does not claim maintainer approval.
