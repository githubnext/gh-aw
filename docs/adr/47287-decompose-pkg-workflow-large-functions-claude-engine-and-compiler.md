# ADR-47287: Decompose Large Functions in pkg/workflow (Claude Engine and Compiler)

**Date**: 2026-07-22
**Status**: Draft
**Deciders**: Unknown (automated refactoring by copilot-swe-agent)

---

### Context

A daily `largefunc` lint scan flagged 5 functions in `pkg/workflow` ranging from 258–373 lines, all exceeding the 60-line limit enforced by the `largefunc` custom linter:

- `ClaudeEngine.GetExecutionSteps` — 373 lines (claude_engine.go)
- `Compiler.generatePrompt` — 284 lines (compiler_yaml_prompt.go)
- `Compiler.buildSafeOutputsJobFromParts` — 271 lines (compiler_safe_outputs_job.go)
- `applyDefaults` — 340 lines (tools.go)
- `parseOnSection` — 258 lines (trigger_parser.go)

These oversized functions mix multiple distinct sub-responsibilities in a single scope — CLI argument construction, environment assembly, prompt import processing, step ordering, and job condition building — making them difficult to read, test in isolation, or extend without risking regressions. The project already established the canonical approach for this class of problem in [ADR-47061](47061-function-length-lint-compliance-via-helper-extraction.md).

### Decision

We will apply the same private-helper-extraction strategy established in ADR-47061 to the 5 offending functions in this PR. Each function's cohesive sub-responsibilities are extracted into focused, single-purpose private helpers named after their responsibility (e.g., `buildClaudeCliArgs`, `resolveClaudePermissionMode`, `buildPreambleTokenSteps`, `calculatePreambleInsertIndex`, `enrichExpressionMappings`). Public interfaces, exported types, and observable behavior are preserved unchanged. A pre-existing bug in `buildCommandTriggerEventsMap` (missing `len(data.LabelCommand) > 0` guard that caused spurious `labeled` event types in `slash_command`-only workflows) is fixed as a side effect of the decomposition.

### Alternatives Considered

#### Alternative 1: Per-function `//nolint:largefunc` suppression

Add inline suppression comments on each of the 5 flagged functions to silence the lint rule without restructuring. This resolves the CI failure instantly but permanently exempts the functions from growth detection, masks future complexity accumulation, and provides no readability or testability benefit. Rejected for the same reasons as in ADR-47061.

#### Alternative 2: Raise the lint threshold for compiler/engine files

Increase the allowed line count above 60 (e.g., 150) for files under `pkg/workflow/` via a linter configuration override. This would pass lint without restructuring, but weakens the quality gate for the entire package and would require repeated increases as functions continue to grow. Rejected because it trades short-term convenience for long-term erosion of the lint contract.

#### Alternative 3: Struct-based decomposition with new intermediate types

Introduce new builder structs (e.g., `claudeCommandBuilder`, `promptImportProcessor`) and attach the extracted logic as methods. This provides stronger encapsulation and enables dependency injection but requires a deeper structural change, risks altering the exported API surface of the `ClaudeEngine` type, and is disproportionate effort for a mechanical lint-compliance refactor of private logic. Rejected as over-engineering for this scope.

### Consequences

#### Positive
- All 5 flagged function-length lint violations are eliminated; the codebase passes `make golint-custom` cleanly for these files.
- Each extracted helper (e.g., `buildClaudeBaseEnvMap`, `applyClaudeTimeoutEnvVars`, `mergeKnownNeedsExpressions`) is independently readable and unit-testable.
- A latent bug in `buildCommandTriggerEventsMap` (spurious `labeled` events in `slash_command`-only workflows) is fixed as a side effect of isolating the guard condition.
- Decomposition of `getExecutionSteps` into `buildClaudeCliArgs` / `buildClaudeCommandString` / `buildClaudeFullCommand` / `buildClaudeCommandEnv` clarifies the pipeline structure of Claude CLI step generation.

#### Negative
- Execution flow is now spread across more function call frames; tracing end-to-end behavior requires following a deeper call graph.
- The extracted helpers are private and tightly coupled to their callers; they carry no reuse benefit today and add navigation overhead when reading unfamiliar code.
- The increased function count in already-large source files (`claude_engine.go`, `compiler_safe_outputs_job.go`, `compiler_yaml_prompt.go`) may require readers to scroll more to find entry points.

#### Neutral
- No changes to public function signatures, exported types, or test files are required.
- The 60-line threshold itself remains unchanged; this ADR addresses compliance with the existing threshold, not the threshold value.
- Future growth in any refactored helper will still be caught by the same `largefunc` lint rule.
- This is the fifth application of the extraction strategy established in ADR-47061 (prior instances: ADR-44008, ADR-36064, ADR-29567, ADR-29721).

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
