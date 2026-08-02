# ADR-49800: Add Dedicated Unit Test File for Activation Step Helpers

**Date**: 2026-08-02
**Status**: Accepted
**Deciders**: Copilot

---

### Context

`pkg/workflow/compiler_activation_steps.go` contains a dense set of activation-job helper methods — including reaction steps, OAuth token checks, lock-file checks, version checks, skill installs, text-output setup, and status comment wiring — many of which return errors. Despite this complexity, none of these helpers had focused unit tests; their behavior was exercised only incidentally through broader compiler-level integration tests. This made it difficult to pinpoint regressions in individual helpers and left error-returning paths (such as sanitization domain computation failures) without direct coverage.

### Decision

We will add a dedicated unit test file (`pkg/workflow/compiler_activation_steps_test.go`) that tests each activation step helper in isolation, using lightweight `activationJobBuildContext` instances constructed directly rather than going through full workflow compilation. Each helper's success path and primary error/skip path will be covered by a focused table-driven or sub-test function.

### Alternatives Considered

#### Alternative 1: Rely solely on existing broad compiler tests

Keep the status quo and cover activation step behavior only through higher-level compiler tests that exercise the full compilation pipeline. This avoids adding a new test file but provides poor regression signal: a failure in any activation helper surfaces as a broad compiler test failure with no indication of which helper broke. Error-returning paths deep in individual helpers are impractical to trigger through the compilation surface.

#### Alternative 2: Add integration-style tests that compile full workflows

Write tests that call `Compiler.Compile(...)` end-to-end with fixture workflow data, then assert on the generated YAML. This would exercise the real pipeline but requires constructing valid full-workflow inputs for every helper scenario, making tests verbose and slow. It also makes it harder to isolate a specific helper's behavior when a test fails.

### Consequences

#### Positive
- Focused tests that directly exercise each helper make regressions in individual methods easy to identify without reading through a full compilation diff.
- Error-returning paths (e.g., malformed model causing sanitization failure) are now explicitly covered, reducing the risk of silent breakage.
- Tests run without a full compilation pass, keeping the unit-test suite fast.
- Writing focused tests surfaced two helpers (`addActivationSkillInstallSteps`, `addActivationSafeOutputMessagesEnv`) whose `error` return types were unreachable; both were removed, simplifying callers and eliminating misleading dead-code paths.

#### Negative
- Adds test code that must be maintained alongside `compiler_activation_steps.go`; renaming helpers or changing their signatures requires updating both files.
- The test helpers (`newActivationStepsTestCompiler`, `newActivationStepsTestContext`) partially duplicate the construction logic of production build contexts and may drift if the context struct evolves.

#### Neutral
- Tests are tagged `//go:build !integration` so they are excluded from integration test runs, consistent with the existing test build tag convention in this package.
- The new file lives in the same `workflow` package (not `workflow_test`), giving it access to unexported types such as `activationJobBuildContext`.

---

*ADR reviewed and accepted as part of PR #49800.*
