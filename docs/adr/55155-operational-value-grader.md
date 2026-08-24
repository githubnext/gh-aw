# ADR-55155: Introduce a Dedicated Operational-Value Grader Type

**Date**: 2026-08-24
**Status**: Draft
**Deciders**: mnkiefer

---

### Context

The gh-aw grader framework already measures execution quality through built-in and custom trace-based graders (coverage, test results, working-set metrics). However, none of those graders can answer whether a workflow run actually achieved its intended repository outcome — e.g., whether the issue assigned to the run was closed, or whether the PR was merged. Operational value is distinct from execution quality: it requires querying accepted, time-bounded repository evidence rather than analysing the run's own trace data. Users need a per-run attainment score in [0, 1] that can be recomputed as evidence matures and optionally compared against a frozen pre-adoption baseline.

### Decision

We will introduce a first-class `operational-value` grader type implemented by a user-authored, deterministic Bash evaluator. The evaluator implements a versioned three-command interface (`--definition`, `--metric`, `--grade-run`); gh-aw verifies its syntax and definition contract at design time via `verify-operational-value-evaluator.sh`, archives the evaluator content with a SHA-256 digest at run time, and executes it in a restricted environment. The primary output is an absolute attainment value in [0, 1]; a frozen baseline value and delta are derived separately and never define the primary value. Regrading a historical run re-downloads the archived evaluator, verifies both digest records match, and recomputes the observation at an explicit evidence timestamp without modifying the original artifact.

### Alternatives Considered

#### Alternative 1: Extend the Existing Custom Grader Interface

Allow custom trace-based graders to optionally return observation metadata alongside their numeric score. This would avoid adding a new grader `source` type and keep the execution path uniform.

Rejected because trace data is the wrong input for operational value: the evaluator must query live (but time-bounded) repository evidence such as issue events or PR state. Mixing live API calls into the trace-grader path would break its hermetic, file-based preprocessing model and make execution order unpredictable. Additionally, conflating execution quality and operational attainment in one result object complicates downstream analysis.

#### Alternative 2: Track Operational Outcomes in an External Observability System

Measure workflow business value in a separate service or database outside the grader framework, and surface results through a dashboard rather than per-run grader results.

Rejected because it requires users to maintain a second system and breaks the unified grader result model that consumers (CI gates, summary tables, the `gh aw graders` CLI) already understand. It also forfeits the digest-verified regrading guarantee, which is essential for auditing value observations over time as evidence matures.

### Consequences

#### Positive
- Enables per-run measurement of true workflow business value using accepted, time-bounded repository evidence.
- Regrading with a digest-verified archived evaluator makes value observations reproducible and auditable across evidence horizons.
- Optional baseline comparison (frozen pre-adoption score) lets teams quantify improvement without redefining the primary metric as a delta.
- The `aw-value` skill gives teams a guided, validated path to designing and verifying their own evaluators.

#### Negative
- The evaluator is user-authored Bash executed with access to `GH_TOKEN`, expanding the trusted-code surface area compared to trace-only graders. The restricted environment and digest check mitigate but do not eliminate this risk.
- A new `source: "operational-value"` grader type requires a separate archiving step, a new execution branch in `trace_graders.cjs`, and a new evaluator interface — all additional maintenance surface.
- Regrading via `gh aw graders operational-value` adds CLI surface area and operational complexity that must be kept in sync with evaluator schema versions.

#### Neutral
- The evaluator interface is versioned (`schemaVersion: 4` for the definition contract, `schemaVersion: 1` for the run request) to allow future evolution without breaking existing evaluators.
- Evaluation is sandboxed to a minimal environment (`PATH`, `HOME`, `GH_TOKEN`, etc.) consistent with other grader execution contexts.
- The `aw-value` skill produces evaluators; the grader framework executes them — the two concerns remain independent.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
