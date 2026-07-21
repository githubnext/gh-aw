# ADR-47088: Split token_usage.go Into Focused Modules

**Date**: 2026-07-21
**Status**: Draft
**Deciders**: Unknown

---

### Context

`pkg/cli/token_usage.go` had grown to approximately 1,100 lines of code serving four distinct responsibilities: token/agent usage file discovery and parsing, AI Credits (AIC) derivation, proxy steering-event classification, and subagent model-request attribution. The resulting monolith made each responsibility harder to evolve safely and likely triggered function-length and file-size linter violations in the project's existing lint rules. All of these concerns operate on shared data models (`TokenUsageSummary`, `TokenCoreMetrics`, etc.) that needed to remain accessible across the responsibilities.

### Decision

We will decompose `pkg/cli/token_usage.go` into six focused files within the same `cli` package, each owning one domain boundary:

- `token_usage_models.go` — shared data models and `TokenUsageSummary` helper methods (`TotalTokens`, `AvgDurationMs`, `ModelRows`)
- `token_usage_parser.go` — token/agent usage file discovery and JSONL parsing, ambient-context extraction
- `token_usage_aic.go` — usage JSONL scanning and AI Credits derivation/population
- `token_usage_steering.go` — proxy event parsing and steering-event classification
- `token_usage_subagent.go` — subagent model-request extraction and request-vs-observed attribution
- `token_usage.go` — reduced to orchestration (`analyzeTokenUsage`, `analyzeTokenUsageAICOnly`) and package-level constants

The existing public API surface and all behavioral semantics (fallback order, steering count integration, per-model aggregation) are preserved without change.

### Alternatives Considered

#### Alternative 1: Retain the monolith with internal grouping

Keep all logic in `token_usage.go` and improve navigability via region comments, blank-line separators, or `//nolint:cyclop` directives to suppress lint. This is the lowest-change option and avoids introducing new files. It was rejected because it does not reduce the cognitive surface — a reader editing AIC logic must still open a file that also contains steering-event and subagent attribution code. It also does not address the root cause of the linter violations.

#### Alternative 2: Move each responsibility into its own sub-package

Create separate packages (e.g., `pkg/cli/tokenusage/parser`, `pkg/cli/tokenusage/aic`) with explicit import boundaries. This would enforce isolation at the compiler level. It was rejected because it would require significant refactoring of the call sites in `token_usage.go` and would make cross-concern access (e.g., `TokenUsageSummary` shared between parser and AIC) more verbose without delivering proportional benefit at this codebase scale. All of the data types are tightly coupled by design.

### Consequences

#### Positive
- Each file is now under 500 LOC and covers a single concern, making it easier to read, test, and extend any one subsystem in isolation.
- Satisfies existing linter rules on function and file length without suppression annotations.
- Clearer module boundaries serve as implicit documentation of the domain model's structure.

#### Negative
- All six files share the same `cli` package, so there is no enforced package-level isolation; a future developer can still mix concerns freely without compiler feedback.
- The number of files to navigate during a code review increases from one to six, requiring reviewers to understand the overall split before they can follow a single data flow.

#### Neutral
- No changes to public API surface or externally observable behavior; callers of `analyzeTokenUsage` and `analyzeTokenUsageAICOnly` are unaffected.
- The decomposition boundary is a convention, not a contract — future contributors should consult this ADR before adding new logic to decide which file is the right owner.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
