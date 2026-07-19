# ADR-46470: Adopt Table-Driven Tests as Canonical Pattern in actionpins Spec

**Date**: 2026-07-18
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

`pkg/actionpins/spec_test.go` had grown with a mix of flat per-scenario test functions and table-driven tests. A testify-expert audit identified several quality problems: the `EnforcePinned` test table covered only failure paths, leaving the happy-path (resolver succeeds) untested; two separate flat functions (`TestSpec_PublicAPI_RecordResolutionFailure` and `TestSpec_PublicAPI_RecordResolutionFailure_DynamicFailed`) tested the same audit behaviour with different inputs, duplicating setup boilerplate; assertion style was inconsistent (`assert.Contains` mixed with `assert.Containsf`); and edge-cases for empty-SHA resolver fallthrough and `nil` `PinContext.Ctx` had no coverage at all. The decision was needed to define the standard test organisation pattern for this package so future contributors follow one consistent approach.

### Decision

We will adopt table-driven tests (`tests := []struct{...}`) as the canonical pattern for any test in `spec_test.go` that validates the same behaviour across multiple inputs or scenarios. Existing flat functions that duplicate setup for scenario variants will be merged into a single table-driven function. New edge-case tests (`TestSpec_DynamicResolution_EmptySHAFallsThrough`, `TestSpec_PublicAPI_ResolveActionPin_NilCtxField`) are added as standalone sub-test functions when they cover a clearly distinct code path rather than a parameter variant. All assertion calls will consistently use the `f`-suffix variants (`assert.Containsf`, `assert.Truef`) to ensure format strings carry the actual values.

### Alternatives Considered

#### Alternative 1: Keep flat per-scenario test functions

Each scenario gets its own top-level `TestSpec_*` function. This is the prior convention used for `TestSpec_PublicAPI_RecordResolutionFailure` and `TestSpec_PublicAPI_RecordResolutionFailure_DynamicFailed`. It is easy to name and find individual cases, but it duplicates setup code, makes it hard to see at a glance which cases are covered, and leads to drift when the shared logic changes (both functions must be updated separately).

#### Alternative 2: Use `t.Run` subtests without a table struct

Group related scenarios under a single parent test using named `t.Run` blocks, each with its own full setup. This avoids the flat-function duplication but does not enforce a shared struct schema, so field names and assertions can diverge between subtests. The table struct approach is strictly better when the inputs and expectations share the same shape, so this alternative is kept only for the genuinely distinct edge-case tests added in this PR.

### Consequences

#### Positive
- Duplicate setup code is eliminated: the merged `TestSpec_PublicAPI_RecordResolutionFailure` function expresses both the no-resolver and failing-resolver scenarios from a single loop.
- Coverage of `resolveActionPinDynamically` is now complete: the previously missing happy-path case (`resolver succeeds with EnforcePinned=true`) is captured in the existing `EnforcePinned` table, and the empty-SHA fallthrough is covered by the new test.
- Assertion messages are richer: `f`-suffix variants print the actual values on failure, reducing the need to re-run tests with added logging.

#### Negative
- Table-driven tests make it slightly harder to isolate a single failing case during interactive debugging because the test loop runs all rows unless the developer uses `-run` with the subtest name.
- Adding a new scenario now requires understanding the table struct shape, which has a mild learning-curve for contributors unfamiliar with the `wantResultSHA` / `wantErrorType` field conventions used here.

#### Neutral
- The two standalone flat tests that were merged cease to exist under their original names; `go test -run TestSpec_PublicAPI_RecordResolutionFailure_DynamicFailed` will no longer match — callers must use the table subtest path instead.
- The new `TestSpec_DynamicResolution_EmptySHAFallsThrough` and `TestSpec_PublicAPI_ResolveActionPin_NilCtxField` are kept as standalone functions (not table rows) because their setup and assertions differ meaningfully from the existing tables.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
