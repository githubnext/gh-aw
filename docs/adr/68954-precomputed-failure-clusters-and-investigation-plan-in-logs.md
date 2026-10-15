# ADR-68954: Precompute failure clusters and a bounded investigation plan inside `gh aw logs`

**Date**: 2026-10-15
**Status**: Draft
**Deciders**: Unknown (inferred from PR #68954)

---

### Context

Agents and humans triaging agentic-workflow failures with `gh aw logs` currently receive a flat list of failed runs and must reconstruct recurring symptoms themselves, typically by reading a very large report (the evaluation in PR #68954 cites a ~4.6 MB JSON report for a 12-hour window with 448 reports and 182 failures). The data needed to group failures — failed jobs and steps, MCP failures, missing tools/data, blocked firewall domains — is already collected and cached per run in the logs JSONL cache. Without grouping, selecting which runs to audit is count-driven and tends to repeatedly sample the same high-volume symptom, while rarer and more informative symptom classes go unexamined. The constraints are that triage must not add network downloads or model calls, must stay deterministic and replayable across report orderings, and must fit inside an agent's limited context budget.

### Decision

We will compute failure clusters and an initial investigation plan deterministically inside the `pkg/cli` logs pipeline, from already-cached run evidence. Observed symptoms are extracted as typed `FailureSignal` values (`pkg/cli/logs_failure_signals.go`), grouped into overlapping `FailureCluster`s with stable content-derived IDs (`pkg/cli/logs_failure_clusters.go`), and reduced to a deduplicated `FailureInvestigation` of at most five runs selected for new-cluster coverage and representative diversity across workflow/engine/model (`pkg/cli/logs_failure_investigation.go`). Clusters are ranked so specific failure boundaries (`failed_job`, `failed_step`, `mcp_failure`) outrank generic failure-handler aftermath and metadata-only unknowns, and intentional failures are excluded from recommendations. Extracted signals are persisted in the JSONL cache and memberships are regenerated per query, so clustering remains a pure, offline, replayable projection of existing data rather than a new data source.

### Alternatives Considered

#### Alternative 1: Leave clustering to the consuming agent

The full logs report already contains every failed job, step, and MCP failure, so an agent could group them itself with no CLI changes, and grouping heuristics could evolve per workflow without shipping Go code. This was rejected because it forces the whole multi-megabyte report into the agent's context before any selection can happen, makes run selection non-deterministic and unauditable, and repeats the same reduction work on every invocation. A deterministic in-CLI projection (~2.8 KB plan) is both cheaper and reproducible, which is what makes the replay test (`TestFailureClustersReplay`) possible.

#### Alternative 2: Rank and select runs purely by cluster size

Keeping the previous count-ranked default and simply reporting the largest symptom groups would be a much smaller change and is trivially explainable. It was rejected because the PR's evaluation shows count-ranking covers only 5 observed clusters within a five-run budget versus 10 under coverage-plus-diversity selection, and it systematically oversamples loud, generic aftermath signals (for example the `Handle agent failure` step) over the specific boundary where the failure originated.

#### Alternative 3: Model-assisted or embedding-based clustering of failure text

Using an LLM or text embeddings over raw log text could group semantically similar failures that do not share a structured signal. It was rejected because it would add model calls and cost to a command whose stated contract is that "clustering itself adds no downloads or model calls", and it would break determinism and replayability. *[TODO: verify whether semantic grouping of metadata-only unknowns is desired as a future, opt-in step.]*

### Consequences

#### Positive
- Triage agents can pick a small, diverse set of runs to audit without reading the full report: the plan is roughly three orders of magnitude smaller than the full JSON report.
- Symptom coverage within a fixed five-run audit budget doubled (5 → 10 clusters) and representative-diversity gaps dropped to zero in the PR's 12-hour evaluation window.
- Clustering is deterministic and order-independent, so results are reproducible and regression-testable (reversed-input replay yields identical clusters and plans).
- Reusing cached evidence paths and audit selectors means no additional API downloads or model calls are introduced.

#### Negative
- Clusters overlap by construction, so their counts cannot be summed to count unique failed runs; every renderer, schema consumer, and prompt must repeat this caveat or readers will miscount failures.
- The approach only groups failures that already emit structured signals; in the evaluation 62 of the analyzed failures remain metadata-only unknowns and gain little triage value.
- Significant new surface area in `pkg/cli` (~900 new lines plus ~600 lines of JSON schema) must be maintained, and the ranking/diversity heuristics are hand-tuned constants (including the budget of five) with no labeled ground truth behind them.
- Signals are persisted in the JSONL cache, so cache entries written by older versions need enrichment shims (`enrichFailureSignal`) and the signal value format becomes a de-facto compatibility surface.

#### Neutral
- Output is surfaced across compact, verbose, console, TSV, JSON, and cross-run renderers; full cluster membership is kept only in JSON and detailed output to limit default verbosity.
- `schemas/logs.schema.json` and `schemas/logs-jsonl.schema.json` grow to describe the new structures, so downstream consumers can project only the fields they need.
- Validation is deterministic consistency, coverage, and guidance checking — not labeled root-cause accuracy or live agent-outcome measurement — so claimed benefits are about selection quality, not diagnostic correctness.
- Documentation and shared prompts (`docs/src/content/docs/troubleshooting/debugging.md`, `.github/workflows/shared/aw-logs-24h-fetch*.md`, `.github/aw/cli-commands.md`) are updated so agents use targeted projections and cached evidence instead of full reports.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
