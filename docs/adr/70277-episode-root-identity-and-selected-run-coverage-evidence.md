---
title: "ADR-70277: Episode Root Identity and Selected-Run Coverage Evidence"
description: Canonical-only episode grouping, recorded root claims, and conservative coverage for selected workflow runs.
---

# ADR-70277: Episode Root Identity and Selected-Run Coverage Evidence

**Date**: 2026-10-19
**Status**: Proposed (maintainer acceptance is not recorded)
**Deciders**: Not recorded

## Context

`gh aw logs` reports a selected set of workflow runs, not a census of an
automation's execution. Date, count, repository, and workflow filters can omit a
root, intermediate caller, or descendant. Calling the earliest selected run a
root misrepresents a descendant as the logical origin. A high-confidence
relationship between selected runs also does not establish that all episode
members were collected.

[ADR-29963](29963-otel-episode-lineage-context.md) introduced propagated episode
and hop identity. Recorded `context.episode_id` can connect selected members
without selecting every intermediate run. Historical call IDs, similar workflow
metadata, and `workflow_run` events do not provide equivalent canonical evidence.
The explicit request to remove legacy episode behavior supersedes this PR's
initial compatibility goal; retaining synthetic episodes or misleading field
aliases is not part of the decision.

The report must distinguish canonical membership, selected-data identity,
recorded root claims, and evidence gaps. These are separate from
[offline outcome accounting](70281-offline-episode-outcome-cost-join.md), which
joins membership, outcome observations, and usage by source run ID and has its
own evidence and compatibility rules.

## Decision

### Canonical membership only

Only runs with a nonempty `context.episode_id` contribute to episode rollups.
Runs with the same canonical ID share an episode even when intermediate runs
are absent. No synthetic episode IDs, call-ID fallback, metadata heuristics, or
legacy bridging reconstruct missing identity. Different canonical IDs never
merge, and an uncorrelated run cannot bridge episodes.

Runs without canonical identity remain available in per-run reporting and
overall totals. Episode totals describe only selected canonical members, so
summing them need not reproduce overall totals. Both compact and verbose JSON
retain every canonical episode, including single-run episodes.

Run-level parent edges require selected, distinct runs with the same canonical
episode ID and no known repository contradiction. Repository identity comparison
is case-insensitive. Reusable workflow hops sharing a GitHub run do not create
self-edges; their recorded repository identity still requires validation.
This is a run-level report, not a per-hop graph.

### Selected identity and recorded roots

Remove episode-level `root_run_id` and `primary_workflow`. The replacement
`representative_run_id` and `representative_workflow` identify selected data,
not aliases for a logical root. Selection prefers the earliest timestamped run,
then the lowest run ID when timestamps are absent or tied. An available selected
workflow name can fill a missing representative workflow name.

`recorded_roots[]` preserves distinct recorded `run_id`, `workflow_id`, and
`repository` claims without substituting a representative. Workflow-only and
repository-only claims remain evidence even when the root run ID is absent.
Original claims are preserved; comparison of repository identities ignores
case. Contradictory populated claims remain visible rather than being resolved
by input order or timestamps.

A root's `observed` flag is true only when its run is selected in that canonical
episode without a known repository mismatch. It does not verify every hop or
artifact. Missing fields stay unknown, and a root is never parsed from an
episode ID. A selected referenced run without canonical identity is unverified
lineage, not evidence that it belongs to a conflicting episode.

### Conservative coverage evidence

Every episode reports `coverage.scope: selected_runs`. Its status is `partial`
when evidence identifies a gap, invalid run ID, contradiction, or applicable
collection truncation; otherwise it is `unknown`. There is no `complete`
episode-coverage state. Even selecting every recorded parent and root cannot
prove that all descendants were observed.

Coverage records missing parent run IDs, preserves root claims, and exposes
machine-readable reasons. Parent validation distinguishes missing runs,
unverified canonical identity, and contradictory identity or repository evidence,
including same-run reusable calls. Lineage confidence remains independent of
coverage: high-confidence canonical membership can have partial coverage.

Collection coverage uses a shared application path. Continuations with known
repository/workflow scope qualify affected episodes rather than unrelated
targets. A reached count limit without sufficient target scope remains
conservative. A truncation flag describes collection evidence, not proof that
this specific episode has additional members; its absence does not establish
completeness.

## Alternatives Considered

### Preserve legacy grouping and root aliases

Keeping call-ID or metadata fallbacks and retaining `root_run_id` as an alias
would reduce migration work. It would also preserve ambiguous membership and
continue presenting selected descendants as roots. Rejected in favor of the
explicit canonical-only requirement and an unambiguous breaking contract.

### Infer completeness from selected roots and parents

Treating a connected selection or exhausted continuation as complete would
simplify downstream reporting. Neither proves that undiscovered descendants,
cross-repository runs, or reusable hops are absent. Rejected; `unknown` is the
honest result when no specific gap is known.

### Resolve root conflicts or discover missing members automatically

Choosing one root claim by timestamp would discard contradictory evidence.
Recursively querying ancestors and descendants, building a per-hop graph, or
maintaining a lifecycle database would introduce discovery, credentials, and
state beyond a selected-run logs report. Rejected for this change; preserving
claims and coverage gaps keeps the report bounded and reproducible.

## Consequences

Canonical grouping matches recorded correlation without inventing lineage.
Consumers can distinguish the selected representative from recorded roots and
inspect incomplete evidence without interpreting confidence as completeness.

This is a breaking output and grouping change. Consumers of removed root fields
must migrate to representative fields for selected-data identity or
`recorded_roots[]` for recorded canonical claims. Historical runs without
canonical identity lose synthetic episode rollups, but retain per-run analysis
and inclusion in overall totals.

Coverage may remain `unknown` even for a valid standalone episode, and conflicting
or incomplete root claims require consumer interpretation. No crawler, per-hop
graph, lifecycle database, or live outcome evaluator is introduced. This decision
does not remove the independent legacy outcome-observation handling or change
the outcome and cost joins described in ADR-70281.

Proposed in [PR #70277](https://github.com/github/gh-aw/pull/70277).
