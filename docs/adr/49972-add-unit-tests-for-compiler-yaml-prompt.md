# ADR-49972: Add Dedicated Unit Tests for compiler_yaml_prompt.go

**Date**: 2026-08-03
**Status**: Draft
**Deciders**: Unknown

---

### Context

`pkg/workflow/compiler_yaml_prompt.go` (336 lines, 11 functions) implements non-trivial prompt-chunking, import-interleaving, and expression-mapping logic for the agentic workflow compiler. Key paths include compile-time inlining vs. runtime-import macro emission, ordered vs. legacy import dispatch, file-read fallback behavior, and expression merge precedence. Despite this complexity, the file had no dedicated test file and relied entirely on indirect integration coverage. A Daily Compiler Code Quality Check (2026-08-03) identified this as the most significant maintainability gap in the package.

### Decision

We will add `pkg/workflow/compiler_yaml_prompt_test.go` with focused unit tests for all 11 functions in the file, exercising each function's happy paths, edge cases (empty inputs, missing files), and behavioral contracts (inline vs. runtime-import macro selection, `all`-entries winning merge precedence over `knownNeeds`). Tests use the `package workflow` internal test pattern (same package, no exported-API surface required) and the `!integration` build tag to keep them out of integration runs.

### Alternatives Considered

#### Alternative 1: Continue Relying on Indirect Integration Coverage

The existing integration tests exercise `compiler_yaml_prompt.go` indirectly through end-to-end compiler calls. This would avoid adding a new test file. However, integration tests run against external dependencies and are slower to iterate on; they also cannot isolate regressions to specific internal functions. Given that 11 functions span multiple non-trivial code paths (ordered vs. legacy dispatch, fallback chains, expression extraction by mode), unit-level coverage provides substantially faster feedback and clearer failure attribution.

#### Alternative 2: Extract Functions into a Sub-Package with Its Own Test File

Moving prompt-chunking helpers into a dedicated sub-package (e.g., `pkg/workflow/prompt`) would allow clean external testing without coupling to unexported symbols. This would add structural churn — renaming and moving a 336-line file, updating all callers — for no functional benefit beyond test isolation. The same behavioral contracts are achievable with `package workflow` internal tests at lower cost.

### Consequences

#### Positive
- Each of the 11 functions in `compiler_yaml_prompt.go` now has explicit behavioral contracts expressed as test cases, making regressions immediately visible without running integration tests.
- Unit tests run without external dependencies and complete in milliseconds, enabling a tight feedback loop during local development.

#### Negative
- The test file couples directly to unexported internal function signatures; any refactor that renames or restructures these functions must update both production code and tests simultaneously.
- Some test cases use implicit zero-count assertions (`wantChunkCount == 0` only fires when `len(promptImports) == 0`), which may leave certain edge cases under-specified in future test extensions.

#### Neutral
- `splitContentIntoChunks` already has dedicated coverage in `xml_comments_test.go`; the new tests exercise it indirectly through the pipeline rather than adding duplicate direct tests.
- The `!integration` build tag aligns with existing conventions in the package for separating unit and integration test runs.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
