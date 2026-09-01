# ADR-55703: Split Over-Length Compiler Functions into Named Helpers

**Date**: 2026-08-25
**Status**: Accepted
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The project enforces a function-length guidance to keep individual functions readable and focused on a single responsibility. A code-quality report identified two functions in the workflow compiler that violated this guidance: `processToolsAndMarkdown` in `compiler_orchestrator_tools.go` (86 lines) and `buildSafeOutputsSetupAndDownloadSteps` in `compiler_safe_outputs_job.go` (91 lines). Both functions had grown organically to inline multiple distinct phases of configuration resolution and step generation, making them harder to read and test in isolation.

### Decision

We will decompose each oversized function by extracting its distinct phases into named helper methods, with intermediate data structs where needed to pass results between phases. `processToolsAndMarkdown` delegates pre-markdown-expansion work to `resolveToolsAndConfig` (returning `toolsAndConfigData`), and `buildSafeOutputsSetupAndDownloadSteps` delegates to three helpers: `buildSafeOutputsSetupSteps`, `buildSafeOutputsDownloadSteps`, and `buildSafeOutputsUserProvidedSteps`. The original functions become thin orchestrators with no behavior change.

### Alternatives Considered

#### Alternative 1: Keep over-length functions as-is

The existing large functions are correct and passing all tests, so doing nothing has zero risk. However, this perpetuates a pattern that conflicts with the project's function-length guidance, makes each function harder to read in isolation, and increases the cognitive load needed to test or extend individual phases.

#### Alternative 2: Move functionality to a dedicated config-resolution service or struct

A more aggressive restructuring — for example, a `CompilerConfigResolver` type owning all configuration phases — would enforce even cleaner boundaries. This is not warranted here because the logic is tightly coupled to the `Compiler` receiver, the PR is purely mechanical, and the larger structural question should be addressed by a separate architectural decision if needed.

### Consequences

#### Positive
- Each extracted helper has a single, stated responsibility and a clear name, improving local readability.
- The orchestrating functions (`processToolsAndMarkdown`, `buildSafeOutputsSetupAndDownloadSteps`) are now thin call-sequence routers, making the overall flow easier to follow.
- Individual helpers are independently testable without exercising the full compilation pipeline.
- Brings both functions back within the project's function-length guidance.

#### Negative
- Introduces `toolsAndConfigData` as a new intermediate struct, adding one more type that callers must understand when reading the code path.
- Slightly deeper call stacks for what are otherwise simple sequential operations; negligible at runtime but adds one more level of indirection when tracing execution.

#### Neutral
- No logic or control flow changes — existing tests remain valid without modification.
- The extracted helpers are package-private and carry no public API implications.

---

*ADR created by [adr-writer agent]. Reviewed and accepted: the implemented changes (`resolveToolsAndConfig`, `buildSafeOutputsSetupSteps`, `buildSafeOutputsDownloadSteps`, `buildSafeOutputsUserProvidedSteps`) match this decision with no logic or control flow changes.*
