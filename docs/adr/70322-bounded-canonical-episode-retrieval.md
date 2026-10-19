# ADR-70322: Bounded Canonical Episode Retrieval from Recorded Episode Identity

**Date**: 2026-10-19
**Status**: Proposed (maintainer acceptance is not recorded)
**Deciders**: Not recorded; maintainer review is pending on PR #70322

---

### Context

`gh aw logs` can group already-downloaded runs into episodes, and the offline `gh aw outcomes episodes` join ([ADR-70281](70281-offline-episode-outcome-cost-join.md)) reports cost per accepted outcome over a *selected-run* snapshot. Users had no supported way to start from one run and collect the other runs of the same episode, which may live in other repositories on the same host. GitHub offers no "runs by episode" query, so any collector must page through run listings and download artifacts — an operation that can consume unbounded REST quota and wall-clock time, and whose discovery is capped by GitHub (10 pages × 100 runs per listing). Episode lineage is already recorded in compact `aw_info` artifacts as `context.episode_id` plus parent/root run references ([ADR-29963](29963-otel-episode-lineage-context.md), [ADR-70277](70277-episode-root-identity-and-selected-run-coverage-evidence.md)), so the problem is retrieval under strict budgets, not identity inference.

### Decision

We will add `gh aw logs episode <run-id-or-url>` (and the equivalent MCP `logs` `episode_run` mode) that collects episode members whose membership is determined **only** by a recorded, matching `context.episode_id`; no synthetic or heuristic grouping is introduced, and conflicting identities are reported as issues rather than resolved. The collector first follows recorded parent/root references, then scans only the seed repository plus explicitly allowlisted (`--allow-repo`) same-host repositories within a fixed creation-time window. Hard budgets — `--max-runs` (distinct candidates attempted, including nonmembers and failed lookups), `--max-api-requests` (every collector-initiated REST call: metadata, discovery pages, artifact inventories, and archive downloads), and `--timeout` — apply across the whole operation, and the output records bounds, stop reasons, and evidence gaps instead of claiming complete membership. Compact info/usage artifacts are cached by host, repository, run, attempt, and immutable artifact ID, with size-guarded ZIP extraction, and results flow through the existing logs report builder and schema.

The per-hop explanation view preserves recorded `hop_id` and `parent_hop_id`
evidence, including reusable-workflow invocations sharing a GitHub run ID.
Missing or conflicting evidence is shown explicitly; neither hop identity nor
edges are synthesized from workflow names or timing. Accounting remains once
per selected run in a separate `run_costs` collection, never allocated to hops.
`gh aw logs episode --view hops` presents the same cached snapshot as the default
summary, without requiring a separate command or JSON file. Identical collection
options reuse a versioned report cache without GitHub calls; `--refresh` rescans
and atomically replaces that snapshot while retaining immutable artifacts.
Collection time and cache provenance remain explicit, and cached request
counters describe the original collection. JSON has one standard logs schema
regardless of the selected display view.

### Alternatives Considered

#### Alternative 1: Heuristic episode reconstruction

Group runs by signals such as workflow name, triggering event, timing proximity, or `workflow_run` chains when no `episode_id` is recorded. This would find more members for older or partially instrumented runs, but it could silently merge unrelated runs and would contradict the canonical-identity approach of ADR-70277 and the evidence-first accounting of ADR-70281. Rejected: false membership is worse than reported gaps for cost-per-outcome reporting.

#### Alternative 2: Unbounded or rate-limit-driven collection

Reuse the existing `logs` model (e.g., `--max-github-api-rate-limit`) and keep scanning repositories and pages until membership appears complete or quota pressure is detected. This is simpler to reason about for small episodes, but cost is unpredictable, cross-repository scans can exhaust shared quota, and GitHub's discovery ceiling means completeness can never actually be proven. Rejected in favour of explicit, user-visible budgets with honest stop reasons.

#### Alternative 3: Resumable paginated collection with stored cursors

Persist page cursors so a later run can continue where a budget-limited run stopped. This would reduce repeated discovery cost, but GitHub run-list pagination is not stable as new runs arrive, so cursors could skip or duplicate candidates. Rejected; larger-bound reruns rescan discovery and rely on the immutable-artifact cache to avoid re-downloading.

### Consequences

#### Positive
- Episode membership is auditable: every member carries a recorded `episode_id`, and conflicts, unavailable runs, and budget stops are surfaced as explicit issues/stop reasons.
- REST usage, candidate count, and duration are strictly bounded per invocation (defaults: seed repo only, last 7 days, 100 runs, 500 requests, 5 minutes), making the command safe for MCP and automation callers.
- Cross-repository collection is opt-in via an allowlist restricted to the seed host, limiting accidental scope expansion.
- Caching by immutable artifact ID makes reruns with larger bounds cheap and avoids trusting mutable names/URLs from API responses.

#### Negative
- Results are explicitly incomplete: runs without recorded `episode_id`, outside the time window, in non-allowlisted repositories, or beyond GitHub's discovery ceiling are never found.
- Reruns repeat discovery API calls because no resume cursor is kept.
- Adds a bounded retrieval path alongside the existing `logs` download pipeline that must be kept consistent with shared report and schema code.
- Budgets count nonmembers and failed requests, so a small `--max-api-requests` can stop before any additional member is found.

#### Neutral
- Only compact info/usage artifacts are downloaded; workflow discovery filters (engine, workflow name, etc.) are rejected in episode mode.
- Artifact-storage redirect transfers are excluded from the REST request count.
- The logs JSON schema gains an episode retrieval section; offline `outcomes episodes` accounting is extended to accept isolated recorded artifact paths.

---

Acceptance remains subject to maintainer review; this document does not record
an approval or change PR merge state.
