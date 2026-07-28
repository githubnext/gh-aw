# ADR-48553: Extract Private Helpers to Reduce Function Length

**Date**: 2026-07-28
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The `pkg/workflow` and `pkg/cli` packages contain numerous functions that exceed the project's function-length lint threshold. As of this PR the backlog stands at 675 findings, with individual functions reaching 377 lines (`BuildAWFCommand`), 259 lines (`generateUnifiedPromptCreationStep`), 226 lines (`buildSafeOutputsSections`), 220 lines (`buildSafeOutputsHandlerOutputsAndActionSteps`), 204 lines (`GetExecutionSteps`), and 199 lines (`checkoutConfigFromMap`). Long functions make it harder to read, reason about, and test individual behaviors in isolation. The lint enforcement creates a growing backlog that compounds with each new feature.

### Decision

We will incrementally reduce the function-length backlog by extracting focused private helper functions from the longest offenders. Each extraction preserves exact logic and introduces no behavioral changes. Helpers are kept private (unexported) in the same file or package to avoid widening the package API. Correctness is verified by recompiling lock files and confirming zero drift.

### Alternatives Considered

#### Alternative 1: Suppress Lint Warnings with `//nolint:funlen` Directives

Add per-function nolint comments to acknowledge exceptions without changing code. This eliminates the backlog instantly with minimal churn. However, it normalizes the long-function pattern, provides no readability or testability benefit, and leaves future maintainers with the same cognitive burden. It was rejected because the lint rule exists precisely to prevent functions from growing unbounded.

#### Alternative 2: Restructure with Design Patterns (Strategy, Visitor, Table-Driven Dispatch)

Replace long imperative functions with data-driven dispatch tables, strategy interfaces, or visitor structs. This approach would eliminate the underlying cause rather than just the symptom, and would provide stronger extensibility. However, it requires non-trivial logic changes and increases refactoring scope and risk significantly — any behavioral divergence is hard to detect in a large diff. It was deferred in favor of the safer, mechanical extraction approach, which can be landed incrementally with high confidence.

### Consequences

#### Positive
- Function-length lint findings reduced (675 → 664 in this PR), demonstrating a repeatable, incremental approach to clearing the backlog.
- Individual private helpers are independently understandable and can be targeted by focused unit tests without executing the entire parent function.
- Smaller call-site bodies reduce the cognitive load for reviewers reading the top-level orchestration logic.

#### Negative
- Call stacks deepen as logic is indirected through private helpers, which can make debugging and profiling slightly harder.
- Granularity decisions are made per-PR rather than against a consistent design document, risking divergent helper naming and abstraction levels across the codebase.
- The backlog is reduced incrementally rather than eliminated; without a systematic plan, the remaining findings may accumulate faster than they are cleared.

#### Neutral
- All extracted helpers remain unexported, so the public API surface of `pkg/workflow` and `pkg/cli` is unchanged.
- Lock file recompilation (`make recompile`) is used as the correctness oracle — zero drift confirms behavioral equivalence.
- The PR specifically calls out one ordering regression (`push_to_pull_request_branch` moved in the safe-output-tools prompt) that was caught and corrected, illustrating that purely mechanical extraction still requires careful review.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
