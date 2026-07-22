# ADR-47250: Classify Driver-Exit (0-Turn) Failures Separately from Agent-Logic Failures

**Date**: 2026-07-22
**Status**: Draft
**Deciders**: Unknown (copilot-swe-agent, pelikhan)

---

### Context

Fleet health dashboards aggregate all failed workflow runs into a single failure count. Some failures occur before the agent ever runs — the CLI wrapper or a pre/post-agent infrastructure step exits non-zero with zero agent turns recorded. These infra-level failures are indistinguishable from genuine agent-logic failures (where the agent ran but the run still concluded as failure), causing health dashboards to overstate fleet ill-health and making it harder to detect actual agent regressions versus transient infrastructure flakiness.

The existing `WorkflowRun` data model already tracks `Turns` (the number of agent conversation turns), which provides a reliable proxy: zero turns means the agent never ran, implying the failure originated in the driver layer.

### Decision

We will add a `driver_exit` vs `agent_logic` failure classifier throughout the health-metrics and logs-report surfaces. The classification rule is: a failed run with `Turns == 0` is a `driver_exit` failure; a failed run with `Turns > 0` is an `agent_logic` failure. This rule is implemented as `isDriverExitFailure(run WorkflowRun) bool` in `pkg/cli/logs_models.go` and propagated to `WorkflowHealth.DriverExitCount`, `WorkflowHealth.AgentLogicFailureCount`, `LogsSummary.TotalDriverExitFailures`, `LogsSummary.TotalAgentLogicFailures`, and the per-run `RunData.FailureKind` field. Operators can filter on `failure_kind` to separate infra flakiness from agent regressions without manually cross-referencing turn counts.

### Alternatives Considered

#### Alternative 1: Operator-Side Filtering (No Schema Change)

Operators could filter `turns == 0` themselves when consuming the existing JSON output, without adding any new fields to the data model. This avoids expanding the public schema.

**Why not chosen**: Every consumer would need to re-derive the classification independently, creating duplicated logic and risk of inconsistency. Embedding the classification in the output makes the distinction first-class and self-documenting for all consumers, including dashboards and automated alerts.

#### Alternative 2: Richer Multi-Category Failure Taxonomy

Instead of a binary `driver_exit`/`agent_logic` split keyed solely on turn count, introduce a broader `failure_category` enum (e.g., `infra_pre`, `infra_post`, `agent_logic`, `agent_timeout`) using additional signals such as log markers, job step names, or exit codes.

**Why not chosen**: Additional signals are not consistently available across all run types and would require deeper changes to the data collection pipeline. The turn-count heuristic is immediately available, simple to verify in tests, and addresses the primary dashboard noise problem without waiting for richer instrumentation.

### Consequences

#### Positive
- Health dashboards no longer conflate infrastructure flakiness with agent regressions; fleet ill-health metrics become more actionable.
- The classification heuristic is simple (`turns == 0`) and fully testable without external dependencies, reducing maintenance burden.
- `failure_kind` is an additive, opt-in field (`omitempty`) so existing consumers that ignore it are unaffected.

#### Negative
- The zero-turn heuristic is a proxy, not a guarantee: a run that fails at the agent's very first action (before completing a turn) would still report `Turns == 0` and be misclassified as `driver_exit` even if the agent did begin executing.
- `failure_kind`, `driver_exit_count`, and `agent_logic_failure_count` are new semi-stable JSON schema fields. Once dashboards and tooling build on them, changing the heuristic or renaming the fields will require a coordinated migration.

#### Neutral
- `WorkflowHealth` gains two new exported fields that appear in JSON output; consumers doing strict schema validation may need minor updates.
- The implementation spans three files (`logs_models.go`, `health_metrics.go`, `logs_report.go`), keeping the classifier logic centralized in `logs_models.go` and consumed by the other two.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
