# ADR-43871: actionpins Spec Tests Verify Behavioral Contracts, Not Structural Type Assertions

**Date**: 2026-07-07
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The `pkg/actionpins` package exposes a public API for resolving and formatting action pin references (SHA passthrough, embedded fallback, dynamic resolution, context propagation, mapping application). Prior spec tests included a category of low-signal assertions that validated struct field values by constructing a type and asserting each field matched what was just set (e.g., `ActionPin`, `ActionYAMLInput`, `ContainerPin`, `ResolutionFailure`). These tests did not exercise resolution logic or behavioral contracts—they could pass even if the resolution functions had serious bugs. Meanwhile, several high-value resolution paths (unknown full-SHA passthrough, invalid mapping skip, `PinContext.Ctx` forwarding to the resolver, `ResolveLatestActionPin` fallback on enforce error) had no coverage.

### Decision

We will remove struct-field-accessor tests that offer no behavioral signal and replace the coverage budget with tests that exercise the resolution API's observable contracts: SHA passthrough format, invalid mapping skip behavior, `PinContext.Ctx` exact forwarding (identity and cancellation state), `ResolveLatestActionPin` fallback-on-error, and enforce-mode failure classification. We will also consolidate repeated enforce-mode sub-tests into a table-driven test and replace manual channel synchronization in the concurrency test with `sync.WaitGroup`.

### Alternatives Considered

#### Alternative 1: Keep Struct-Field Tests and Add Behavioral Tests on Top

Preserve all existing struct-field assertions and add the new behavioral tests alongside them. This would increase total test count without reducing noise.

Not chosen because it retains low-signal tests that obscure the spec's intent, increase maintenance cost when type shapes evolve, and provide false confidence. The signal-to-noise ratio of the test file degrades; reviewers must sift through field-assertion boilerplate to find the behavioral contracts.

#### Alternative 2: Elevate to Integration Tests at a Higher Call Level

Replace unit-level public-API tests with integration-style tests that exercise the full resolution pipeline from an end-to-end workflow fixture (e.g., a real GitHub Actions workflow YAML with mixed pinned and unpinned references).

Not chosen because integration tests at the workflow level would be slower, harder to isolate, and would not exercise the specific behavioral branches (exact context forwarding, invalid mapping, fallback-after-error) as precisely. The public API surface of `actionpins` is the right contract boundary for specification tests.

### Consequences

#### Positive
- Tests now document and enforce observable behavioral contracts (what the API does) rather than structural facts (what types look like), making them more likely to catch real regressions.
- Table-driven refactor of `ResolveActionPin_EnforcePinned` reduces repetition and makes adding new enforce-mode scenarios a one-line struct literal.
- `sync.WaitGroup` pattern in the concurrency test is idiomatic Go and removes the need for a goroutine-count channel that could theoretically produce false positives.
- Newly covered resolution paths (SHA passthrough, mapping skip, context forwarding, fallback-on-error) close gaps that previously allowed silent behavioral regressions.

#### Negative
- Struct type shapes (`ActionPin`, `ActionYAMLInput`, `ContainerPin`, `ResolutionFailure`) are no longer explicitly validated in tests; if a field is renamed or removed, only the behavioral tests that happen to use that field will catch it.
- The test helper `testSHAResolver` now captures call arguments (`capturedCtx`, `capturedRepo`, `capturedRef`), making it a stateful stub; this works for single-call tests but would require reset logic for multi-call scenarios.

#### Neutral
- The test file line count changes from more lines of lower-signal assertions to fewer, more precise behavioral assertions — net delta is still a modest addition because new coverage was added.
- `require.NotEmpty` replaces `assert.NotEmpty` at points where follow-on assertions depend on a non-empty result; this is a test-correctness improvement that prevents misleading assertion failures on nil dereference but does not change the behavioral contract being tested.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
