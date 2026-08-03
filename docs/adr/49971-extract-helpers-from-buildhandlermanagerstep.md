# ADR-49971: Extract Helper Methods from buildHandlerManagerStep

**Date**: 2026-08-03
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

`buildHandlerManagerStep` in `pkg/workflow/compiler_safe_outputs_steps.go` was a ~250-line function that mixed three distinct responsibilities: (1) minting per-handler and dispatch-repository GitHub App token steps, (2) assembling core environment variables (agent output reference, allowed domains, URL policy, custom handler registration maps), and (3) assembling token environment variables (CI-trigger token, project tokens, assign-to-agent token, agent-session token, cross-repo GITHUB_TOKEN override). The function scored 74/100 in the daily compiler code quality check, with oversized single-function structure flagged as the primary concern. The mixed concerns also made it impossible to unit-test sub-behaviors in isolation.

### Decision

We will extract `buildHandlerManagerStep` into three focused helper methods on `*Compiler`: `addAppTokenMintingSteps`, `addSafeOutputCoreEnvVars`, and `addSafeOutputTokenEnvVars`. The main function is reduced to ~41 lines of pure orchestration that calls these helpers in sequence. No behavior changes are made; this is a pure refactor that preserves identical generated YAML output.

### Alternatives Considered

#### Alternative 1: Keep the monolithic function as-is

The existing function is logically correct and all tests pass, so deferring the refactor avoids churn. However, this leaves the three sub-behaviors untestable in isolation and the 74/100 code quality score unaddressed. Rejected because improving testability and maintainability outweigh the cost of a well-scoped, behavior-preserving refactor.

#### Alternative 2: Extract to a dedicated builder struct

A separate `safeOutputStepBuilder` type could encapsulate all three responsibilities behind a richer API and stronger encapsulation boundaries. However, this would require introducing a new type, refactoring call sites, and expanding the scope significantly beyond what the problem warrants. Rejected as over-engineering for a pure function decomposition.

### Consequences

#### Positive
- `buildHandlerManagerStep` is now a 41-line pure orchestration function that clearly expresses the step-construction sequence.
- 20 new unit tests directly exercise the three extracted helpers (`TestAddAppTokenMintingSteps`, `TestAddSafeOutputCoreEnvVars`, `TestAddSafeOutputTokenEnvVars`), enabling targeted regression detection for each concern.
- The compiler code quality score for the file is expected to improve beyond 74/100.

#### Negative
- `addSafeOutputCoreEnvVars` and `addSafeOutputTokenEnvVars` accept a `*[]string` pointer parameter rather than returning new slices, which is less idiomatic Go and requires callers to pass the address of their local slice.
- Three additional methods are added to the `Compiler` type, increasing its method surface area (though all three are unexported).

#### Neutral
- The extracted helpers live in the same file (`compiler_safe_outputs_steps.go`); no new files or packages are introduced.
- All pre-existing tests pass unchanged — behavior preservation is enforced by the existing test suite.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
