# ADR-49793: Extract Workflow Engine Execution Helpers for largefunc Compliance

**Date**: 2026-08-02
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The `gh-aw` codebase enforces a custom `largefunc` lint rule capping function length at 60 lines. The workflow engine `GetExecutionSteps` methods across `AntigravityEngine`, `BehaviorDefinedEngine`, `CodexEngine`, and `PiEngine` had grown to 139–318 lines. Similarly, log-parsing functions `parseClaudeJSONLog` (Claude) and `ParseLogMetrics` (Copilot) reached 180–185 lines each. These 24 violations blocked the lint gate in CI. The remediation must be scoped to the engine execution-builder and log-parser paths only, without changing any external behavior or public signatures.

### Decision

We will decompose oversized `GetExecutionSteps` implementations and log-parser functions into cohesive private helper methods on the same receiver type. Each helper encapsulates one phase: argument construction, firewall/AWF command wrapping, allowed-domain resolution, environment assembly, or step rendering. The public function becomes a thin orchestrator delegating to these helpers. Log parsers are split into entry collection, format-detection, and metric-extraction helpers following the same pattern.

### Alternatives Considered

#### Alternative 1: Raise or Disable the largefunc Lint Limit

Increase the `largefunc` limit (e.g., to 120 or 200 lines) or add per-file lint exemptions. This avoids code churn and keeps each function's full logic visible in a single read, reducing the risk of introducing subtle bugs during the split.

This was rejected because the lint limit exists to enforce long-term maintainability. Raising or exempting it for specific files creates inconsistent enforcement and allows technical debt to accumulate unchecked as engines grow.

#### Alternative 2: Extract Phases into Dedicated Builder Types or Separate Files

Create separate struct types (e.g., `AntigravityExecutionBuilder`) or move each phase into its own file, decoupling phases from the engine receiver entirely.

This was rejected as unnecessary abstraction for what is a localized refactor. The engines are already well-scoped types; adding private methods on the same struct is idiomatic Go and avoids introducing additional indirection layers that would require callers to understand a new type hierarchy.

### Consequences

#### Positive
- All targeted `largefunc` violations are cleared, unblocking the CI lint gate.
- Each extracted helper has a single, clearly named responsibility (e.g., `buildAntigravityExecutionEnv`), making individual phases easier to locate and modify in isolation.
- Smaller functions are independently testable if unit test coverage is expanded in the future.

#### Negative
- Each engine struct now exposes a larger surface of private methods, making it harder to get an overview of the struct at a glance.
- The additional call-graph depth can make debugging stack traces marginally harder to read.

#### Neutral
- No external API or behavioral change: `GetExecutionSteps` signatures remain identical and all callers are unaffected.
- The same decomposition pattern was applied uniformly across four engines and two log parsers, yielding a consistent shape across the `pkg/workflow` package.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
