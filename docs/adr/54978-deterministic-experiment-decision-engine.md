# ADR-54978: Deterministic Experiment Decision Engine

**Date**: 2026-08-23
**Status**: Accepted
**Deciders**: Unknown

---

### Context

The `gh aw experiments analyze` command previously computed statistical metrics (p-values, effect sizes, Mann–Whitney ranks) and emitted a two-state `recommendation` field (`EXTEND` or `READY_FOR_ANALYSIS`). This left the promotion verdict — should we advance the candidate variant? — entirely to human judgment or custom downstream scripts. As automation of promotion workflows becomes a goal, the analysis pipeline needs to emit a stable, machine-readable verdict that future tooling can consume without re-interpreting raw statistics or re-downloading grader artifacts. The verdict must also respect guardrail metrics (secondary health checks that can veto promotion regardless of primary metric improvement) and support both frequentist (p-value) and Bayesian (probability of superiority) evidence models.

### Decision

We decided to introduce a pure, stateless `DecideExperiment(ExperimentAnalysis) ExperimentDecisionResult` function that transforms an existing analysis result into one of four standardized decisions — `EXTEND`, `PROMOTE`, `REJECT`, or `INCONCLUSIVE` — using a configurable per-experiment policy (`minimum_effect`, `regression_tolerance`, `confidence`). The decision layer is embedded in the existing analysis pipeline as the last step and emits `decision`, `reason_code`, `decision_policy`, and `decision_guardrails` fields in JSON output. It performs no I/O, does not re-run statistical tests, and does not mutate experiment state.

Readiness remains a separate contract: `COLLECTING` means at least one variant has fewer than `min_samples` usable observations, while `READY` means the primary sample gate is met. Readiness does not encode the outcome. In particular, a `READY` experiment can be `PROMOTE`, `REJECT`, `INCONCLUSIVE`, or—when a mandatory guardrail or analysis input is unavailable—`EXTEND`. The legacy `recommendation` field remains `EXTEND` / `READY_FOR_ANALYSIS` for compatibility.

For exactly two variants, the policy applies sample readiness, mandatory guardrail availability, guardrail failure, material regression, material improvement, and the inconclusive fallback in that order. Missing guardrail evidence extends collection and a failed guardrail rejects before the primary metric can promote. More than two variants are analyzed statistically but receive `INCONCLUSIVE` / `unsupported_multi_variant` rather than an automatically selected winner.

Effects use absolute primary-metric units. The engine normalizes `direction: min` and `direction: max` so a positive normalized effect always favors the candidate. `minimum_effect` defaults to zero, `regression_tolerance` defaults to `minimum_effect`, and `confidence` defaults to 0.95. Frequentist evidence uses `p <= 1-confidence`; Bayesian evidence uses probability of superiority directly.

The stable reason-code vocabulary is `insufficient_samples`, `insufficient_observations`, `candidate_improved`, `candidate_regressed`, `guardrail_failed`, `guardrail_unsupported`, `effect_below_threshold`, `evidence_insufficient`, `unsupported_multi_variant`, and `analysis_unavailable`. The JSON contract also carries control/candidate identities, sample counts, direction, effect, and evidence when available. Future optimizer or promotion tooling must consume this result rather than reinterpret statistics; automatic promotion, traffic mutation, and candidate generation remain outside this decision.

### Alternatives Considered

#### Alternative 1: Extend the existing `recommendation` field with additional states

The `recommendation` field (`EXTEND`, `READY_FOR_ANALYSIS`) could have been extended to include `PROMOTE`, `REJECT`, and `INCONCLUSIVE`. This would avoid adding a new top-level field. Rejected because `recommendation` conflates sample-readiness (a pre-analysis gate) with the post-analysis verdict (a policy interpretation). Extending it would break existing consumers that parse the field for readiness only, and the field carries no policy context (minimum effect, confidence threshold) needed for automation to audit the verdict.

#### Alternative 2: A separate `gh aw experiments decide` command that re-fetches artifacts and re-runs statistics

A new command that independently downloads run artifacts and re-runs statistical tests before applying a decision policy. Rejected because it would re-download expensive grader artifacts (potentially gigabytes), could produce results that diverge from the original analysis due to data timing, and violates the stated design constraint (R-STAT-017) that the decision layer must consume existing analysis results and must not re-fetch observations or rerun tests.

### Consequences

#### Positive
- Provides a stable, machine-readable JSON contract (`decision`, `reason_code`, `decision_policy`, `decision_guardrails`) that future promotion automation can consume without re-interpreting raw statistical artifacts.
- The decision function is a pure transformation with no I/O or side effects, making it independently unit-testable and reproducible given the same analysis input.
- Guardrail evaluation now resolves per-variant aggregates (mean, pass/fail) from grader and eval observations, enabling mandatory health checks to veto promotion regardless of primary metric outcome.
- Separating `COLLECTING` / `READY` from the four decision states prevents sample availability from being mistaken for an experiment outcome.

#### Negative
- A policy configuration change (e.g., raising `minimum_effect`) does not retroactively update previously recorded decisions; the full analysis pipeline must be rerun to produce a decision under the new policy.
- The decision layer cannot distinguish a misconfigured native metric name from a legitimately missing observation; both cases silently return `EXTEND` with `reason_code: insufficient_observations`, which may mask configuration errors.
- Automatic adjudication is limited to exactly two variants; multi-variant comparisons remain descriptive.

#### Neutral
- Guardrail observation resolution expands the artifact download scope: previously, only primary metric grader artifacts were fetched; now guardrail metric artifacts are also resolved and downloaded when guardrail metrics are configured.
- The `ExperimentDecisionResult` is embedded in `ExperimentAnalysis` via struct embedding, so all existing JSON consumers receive the new decision fields automatically alongside existing analysis fields.
- Simultaneous-experiment interaction diagnostics remain a reporting concern. The daily report may hold a core `PROMOTE` as presentation-level `EXTEND` when factorial cells are underpowered, but it preserves the core decision.

---
