# ADR-45333: Extract `buildMainJob` Helpers Into a Focused Helpers File

**Date**: 2026-07-14
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

`buildMainJob` in `pkg/workflow/compiler_main_job.go` had grown to 420 lines — 7× over the project's 60-line `largefunc` threshold. The function mixed five distinct concerns in a single body: job-condition construction, dependency resolution, output declaration, environment variable setup, and permission inference. This made the function difficult to understand in isolation, impossible to unit-test at the helper level, and a frequent source of merge conflicts as each concern evolved independently. The project enforces a maximum function length via a static linter; the monolith triggered this check on every CI run.

### Decision

We will extract the five logical concerns from `buildMainJob` into nine focused, single-responsibility helper methods, placing them in a new sibling file `pkg/workflow/compiler_main_job_helpers.go` within the same `workflow` package. The orchestrating `buildMainJob` function will be reduced to under 60 lines, delegating to the helpers via direct method calls. A parallel test file `compiler_main_job_helpers_test.go` will provide 22 targeted unit tests that exercise each helper boundary independently.

### Alternatives Considered

#### Alternative 1: Keep the Monolith and Suppress the Linter

The 420-line function could be left as-is, with a per-file or per-function linter suppression comment to silence the `largefunc` threshold violation. This avoids structural change and eliminates the refactor risk.

Not chosen because suppressing the linter validates a known code quality problem and sets a precedent for bypassing the threshold. The function's mixed concerns make it genuinely harder to test and review; suppression papers over the symptom without addressing readability or testability.

#### Alternative 2: Introduce a Dedicated `MainJobBuilder` Struct

The helpers could be grouped into a new `MainJobBuilder` type with its own constructor and field state, rather than remaining methods on the existing `Compiler` type. This would provide stronger encapsulation and clearer ownership of main-job-specific state.

Not chosen because the helpers all read from `Compiler` receiver state and `WorkflowData` — introducing a new type would require passing or copying that state across a boundary, adding boilerplate without a clear benefit at this scale. The helpers can be promoted to a dedicated type in a future pass once the right ownership boundary becomes clearer.

### Consequences

#### Positive
- `buildMainJob` reduced from 420 lines to under 60; the function now reads as a concise orchestration sequence.
- Each helper function covers a single concern and can be unit-tested in isolation; 22 new tests were added covering condition variants, dependency deduplication, output structure, env initialization, and permission inference.
- The linter `largefunc` violation is resolved without suppression, keeping CI green for all future PRs.
- Smaller, named helpers make code review and future modification scoped to one concern at a time.

#### Negative
- Understanding `buildMainJob` fully now requires navigating to a second file (`compiler_main_job_helpers.go`), adding one level of indirection.
- The helper file is 381 lines itself; if concerns continue to grow, a second round of extraction or a dedicated type may be needed.

#### Neutral
- No public API changes — all extracted helpers are unexported package-level or receiver methods.
- Import surface of `compiler_main_job.go` shrinks (several packages moved to the helpers file), which marginally reduces compilation coupling on the main file.
- The refactor is purely internal; generated workflow YAML output is unchanged.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
