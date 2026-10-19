# ADR-70205: CLI-Only Outcome Evaluation and Explicit Session Lineage

**Date**: 2026-05-14
**Status**: Draft
**Deciders**: pelikhan, Copilot coding agent [TODO: verify]

---

> **Historical proposal replaced before merge:** The durable pending-backlog design
> was not the final decision in [#70205](https://github.com/github/gh-aw/pull/70205),
> merged on 2026-10-19. The merged change removed the scheduled collector and
> duplicate JavaScript evaluators in favor of CLI-only evaluation through
> `gh aw outcomes`. This ADR remains Draft; the implementation evidence below
> does not establish formal ADR acceptance.

### Context

Before this change, the outcome collector (`actions/setup/js/evaluate_outcomes.cjs`) discovered recent successful runs, downloaded their `safe-outputs-items` manifest, evaluated each safe-output item, and recorded the run ID in a "seen" cache (a flat array capped at 500 entries). Every evaluated run was marked seen even when its outcomes were still `pending`, `unknown`, or `error`. Such runs were never revisited, so their eventual outcomes could be lost. Read and listing failures could appear as "no data", dispatch run IDs and cross-repository PR attribution could be dropped, legacy episode lineage could be merged ambiguously, and unified sessions carried no structured parent identity from which to rebuild episodes.

### Final merged decision

The merged PR made `gh aw outcomes` the sole live-evaluation entry point. Ordinary workflow
execution records safe-output evidence rather than deciding acceptance. Fresh and cached
audit paths no longer evaluate outcomes or resurface historical outcome snapshots. Later
`gh aw outcomes` invocations reevaluate current GitHub state using the available evidence.

The scheduled Outcome Collector, duplicate JavaScript evaluators, collector-only OTLP span
export, and obsolete collector tests were deleted. No built-in pending backlog, scheduled
reevaluation, or collector-cache migration shipped as the resolution to this proposal.
The manual objective impact report invokes `gh aw outcomes --json` and projects that output
into its datasets rather than maintaining a second classification implementation.

The merged change retained explicit parent lineage (`parentRepo`, `parentRunId`,
`parentRunAttempt`, `parentWorkflowId`, `parentWorkflowCallId`, `parentEventType`) projected
from `aw_context` into unified session episode metadata. Recorded caller identity remains
separate from current-run identity. Dispatch run IDs, including numeric-string IDs, are
preserved, and agent-linked PRs are evaluated in the repository recorded in the issue's
`repository_url`. Episode lineage conflicts are resolved deterministically, without
merging disconnected legacy `workflow_call` runs or inferring missing lineage from revisions.

Implementation evidence is in `pkg/cli/outcomes_command.go`, the Go outcome evaluators,
`pkg/cli/logs_episode.go`, `actions/setup/js/unified_session_payload.cjs`, and the objective
impact report. The 82-case conformance corpus in
`pkg/cli/testdata/outcome_conformance.json` remains for the Go evaluator; it is no longer a
shared Go/JavaScript evaluator contract.

### Historical alternatives considered before merge

#### Alternative 1: Keep the single seen cache and skip unresolved runs

Keeping the old collector behavior would have marked every evaluated run seen regardless
of status. This bounded state but could lose outcomes that resolved later. The merged
decision removed the collector rather than retaining this policy.

#### Alternative 2: Re-scan the full run history every cycle

Rescanning a wider window every collector cycle would have avoided extra state at increasing
API cost, without recovering expired manifests. This collector-based alternative did not
ship; CLI reevaluation still depends on evidence being available.

#### Alternative 3: Infer missing lineage from revision metadata

Inferring parentage from matching repository, SHA, or ref would have avoided a schema change
but could merge unrelated invocations. The merged change retained explicit parent identity
and conservative handling of missing lineage.

### Consequences of the merged decision

#### Positive
- Live classification is centralized in the CLI instead of duplicated in a scheduled collector and JavaScript evaluators.
- Pending, unknown, or error outcomes can be reevaluated by a later CLI invocation; they are not promised automatic reevaluation.
- Unified sessions carry recorded parent identity without inferring missing relationships. Older sessions without lineage remain valid.
- Cross-repository agent PRs are evaluated in the correct repository, and dispatch run IDs are preserved.

#### Negative
- There is no built-in scheduled reevaluation or outcome OTLP exporter.
- Later evaluation requires another CLI invocation and retained evidence or downloadable artifacts; the removed backlog does not guarantee access after artifact expiry.
- The unified session schema and both summary renderers gained parent fields that must stay in sync.

#### Neutral
- AIC per accepted outcome is computed from observed full-run cost (`TotalAIC / Accepted`), not from per-item cost attribution.
- Episode usage rollups and outcome summaries remain separate.

### Superseded proposal: durable pending backlog (not shipped)

The initial revision proposed a versioned resolved-run cache (`seen-runs.json`,
`{version: 2, ids}`), a pending backlog (`pending-runs.json`), and copied manifests at
`run-<id>/safe-output-items.jsonl`. Unresolved runs would have been revisited outside the
discovery window, legacy seen arrays discarded once, and corrupt state treated as an error.
It also proposed collector counters such as `runs_revisited` and `unresolved_runs`.

Those cache formats, migration rules, retention implications, and revisit guarantees describe
only the abandoned collector design. They are not supported behavior or implementation
requirements for `gh aw outcomes`.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
