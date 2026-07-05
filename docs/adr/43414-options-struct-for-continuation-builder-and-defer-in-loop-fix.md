# ADR-43414: Options Struct for Continuation Builder and Defer-in-Loop Fix in pkg/cli

**Date**: 2026-07-05
**Status**: Draft
**Deciders**: Unknown (automated lint compliance via Copilot SWE agent)

---

### Context

Two mechanical lint violations existed in `pkg/cli`:

1. `buildContinuationIfNeeded` in `logs_orchestrator.go` accepted 11 positional parameters (`workflowName`, `startDate`, `endDate`, `engine`, `branch`, `afterRunID`, `count`, `timeoutMinutes`, plus three booleans). This violated the project's parameter-count lint rule and made call sites error-prone because two same-type arguments (e.g., adjacent `string` parameters) can be silently transposed with no compiler error.

2. `waitForServerReady` in `mcp_inspect_mcp_scripts_server.go` registered a `defer resp.Body.Close()` inside a polling loop. Per the `deferinloop` linter rule (see [ADR-40679](40679-add-deferinloop-linter.md)), deferred calls inside a loop do not execute until the enclosing function returns, not at the end of each iteration — potentially accumulating open response bodies.

The bulk of the line count in this PR (1512 of 1599 additions) is a JSON reformatting of `pkg/actionpins/data/action_pins.json` and `pkg/workflow/data/action_pins.json` from 2-space to 4-space indentation; this is cosmetic with no functional impact and does not represent a design decision.

### Decision

We will introduce a `continuationOptions` struct to group the eight configuration parameters of `buildContinuationIfNeeded`, reducing the positional argument count to three meaningful booleans plus one options value. We will extract an `isServerReady` helper so that `resp.Body.Close()` is deferred within the helper's scope rather than the polling loop body. Both changes preserve existing behavior while bringing the code into compliance with project lint rules established in [ADR-40679](40679-add-deferinloop-linter.md) and the parameter-count linter.

### Alternatives Considered

#### Alternative 1: Suppress lint violations with `//nolint` directives

Add inline `//nolint:paramcount` and `//nolint:deferinloop` comments to suppress the findings in-place without changing the code. This is the lowest-effort option and leaves the function signatures untouched. It was not chosen because suppression defeats the purpose of these lint rules — they enforce maintainability invariants that the project explicitly adopted via ADRs — and would set a precedent for bypassing them in newly written code.

#### Alternative 2: Close the response body explicitly (without defer) inside the loop

Replace `defer resp.Body.Close()` with an immediate `resp.Body.Close()` at the point where the success branch returns, keeping the defer-in-loop fix without extracting a helper function. For the parameter-count issue, defer changing the signature by splitting `buildContinuationIfNeeded` into two smaller focused functions instead of grouping parameters into a struct. These options were not chosen because the helper extraction makes the readiness check self-contained and reusable, while the struct approach is already established in this codebase (see [ADR-31336](31336-refactor-download-workflow-logs-to-options-struct.md)) and is the idiomatic Go solution for grouping related parameters.

### Consequences

#### Positive
- Call sites for `buildContinuationIfNeeded` are now self-documenting: named struct fields make the mapping from variable to parameter explicit, eliminating silent transposition bugs.
- Response bodies in `waitForServerReady` are closed immediately at the end of each successful polling iteration via the helper's deferred close, removing the risk of resource accumulation across iterations.
- The code passes the project's `deferinloop` and parameter-count lint rules without suppressions, keeping lint signal clean for future contributors.

#### Negative
- Callers of `buildContinuationIfNeeded` must construct a `continuationOptions` struct literal instead of passing positional arguments; existing call sites required mechanical updates.
- The `continuationOptions` type is unexported (`continuationOptions` vs `ContinuationOptions`), limiting it to `pkg/cli` — if the function is later promoted or tested from outside the package, the type must be exported at that point.

#### Neutral
- Test files (`logs_orchestrator_unit_test.go`) required mechanical refactoring to use struct literal syntax; no test behavior changed.
- JSON indentation in `pkg/actionpins/data/action_pins.json` and `pkg/workflow/data/action_pins.json` changed from 2-space to 4-space; no functional impact.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
