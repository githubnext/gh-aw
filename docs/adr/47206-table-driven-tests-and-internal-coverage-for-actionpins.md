# ADR-47206: Table-Driven Tests and Internal Coverage for actionpins

**Date**: 2026-07-22
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The `pkg/actionpins` package is a core library responsible for resolving, mapping, and pinning GitHub Actions references and container images. Its internal test file (`actionpins_internal_test.go`) had grown a set of per-scenario test functions for `applyActionPinMapping` (six separate `TestApplyActionPinMapping_*` functions), each with duplicated `PinContext` setup boilerplate. Additionally, several internal helper functions — `getLatestActionPinReference`, `recordPinResolutionFailure`, `logDynamicResolutionSkipped`, and the container-pin branch of `loadActionPinsData` — had no direct test coverage, leaving edge-case nil-safety and boundary behaviors unverified. The PR #47206 was opened to close these gaps and improve test maintainability.

### Decision

We will consolidate the `applyActionPinMapping` per-scenario tests into a single table-driven `TestApplyActionPinMapping` function, and add four new internal test functions targeting previously uncovered code paths (`TestLoadActionPinsData_LoadsContainerPins`, `TestGetLatestActionPinReference_ReturnsFormattedReferenceOrEmpty`, `TestLogDynamicResolutionSkipped_NoResolverBranch`, `TestRecordPinResolutionFailure_NilSafety`). The table-driven approach is chosen because it is idiomatic Go, reduces boilerplate for each new scenario, and surfaces all cases in a single view, making it easier to audit coverage.

### Alternatives Considered

#### Alternative 1: Continue Adding Per-Scenario Test Functions

The status quo approach — each mapping scenario gets its own top-level `TestApplyActionPinMapping_*` function. Why considered: matches the existing style and requires no refactoring. Why not chosen: the six existing functions already contain significant boilerplate duplication; adding further scenarios would compound maintenance cost and make it harder to see at a glance which cases are covered. The `repeat` field needed for the deduplication scenario is also awkward to express without the table struct.

#### Alternative 2: Move Missing Coverage to Black-Box Spec Tests in spec_test.go

Add coverage for `getLatestActionPinReference`, `recordPinResolutionFailure`, and `logDynamicResolutionSkipped` in the `actionpins_test` (black-box) package rather than the internal test package. Why considered: black-box tests are less brittle because they don't depend on unexported symbols. Why not chosen: the functions under test are unexported (`getLatestActionPinReference`, `recordPinResolutionFailure`, `logDynamicResolutionSkipped`); they cannot be called from `actionpins_test`. Internal tests are the only way to exercise them directly, and the nil-safety behavior is only observable via those unexported entry points.

### Consequences

#### Positive
- Adding new `applyActionPinMapping` scenarios now requires adding a single struct literal to the test table rather than a new top-level function, lowering the per-scenario authoring cost.
- Previously untested nil-safety and no-resolver branches are now explicitly verified, reducing the risk of panics in production edge cases (nil `PinContext`, nil `RecordResolutionFailure` callback).
- The container-pin branch of `loadActionPinsData` is now independently verified without relying on the embedded data fixture.
- A package-level doc comment added to `spec_test.go` clarifies the purpose of the black-box test module for future contributors.

#### Negative
- Table-driven tests with a large number of fields (`name`, `actionRepo`, `version`, `mappings`, `repeat`, `wantRepo`, `wantVersion`, `wantMappingNotification`, `wantMapNotificationKeys`) can be harder to scan when a single case fails — the failure message must be read alongside the struct to reconstruct the intent.
- The `repeat` field is non-obvious: it exists solely for the deduplication scenario; all other cases default to `repeat = 1` via `max(tt.repeat, 1)`. This implicit default must be understood by future editors.

#### Neutral
- No production code was changed; test-only changes do not affect binary artifacts or runtime behavior.
- The consolidation deletes 79 lines and adds 181, for a net increase of 102 lines — the expanded table carries more information than the removed boilerplate.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
