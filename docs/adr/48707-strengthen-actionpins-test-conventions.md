# ADR-48707: Strengthen Test Conventions in actionpins_internal_test.go

**Date**: 2026-07-28
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

`pkg/actionpins/actionpins_internal_test.go` had accumulated several quality gaps over time: panic assertions used `assert.Panics` (which only checks that a panic occurred, not what it says), loop-based test iterations did not use `t.Run` subtests (so a single failure aborted all remaining assertions), four related top-level functions duplicated nearly identical setup for `resolveExactHardcodedPin`, and a shared helper type was defined mid-file between unrelated tests. These gaps meant that message-changing refactors could silently pass, and a single flaky image would mask all others in the container-pin loop.

### Decision

We will refactor `pkg/actionpins/actionpins_internal_test.go` to adopt four specific testing conventions: (1) use `assert.PanicsWithValue` instead of `assert.Panics` to pin the exact panic message string; (2) wrap loop iterations in `t.Run` subtests so each item is reported independently; (3) consolidate structurally-identical top-level test functions for the same subject into a single table-driven test; and (4) position shared helper types near the top of the file, immediately after imports.

### Alternatives Considered

#### Alternative 1: Preserve existing test structure and only add missing coverage

Add new test cases on top of the existing pattern without restructuring. This avoids changing existing test signatures but perpetuates weak panic assertions and loop-based iterations. Future contributors would continue to follow the weaker pattern. Rejected because it leaves the root quality gaps in place and increases divergence from patterns already used elsewhere in the file.

#### Alternative 2: Extract shared test helpers into a separate `_test_helpers_test.go` file

Move `countingResolver` and any future helpers into a dedicated helper file to keep the main test file focused on test cases only. This is a valid long-term approach for large test suites but adds file-management overhead for what is currently one small helper type. Rejected as premature; relocating the helper to the top of the existing file achieves the same readability goal with less churn.

### Consequences

#### Positive
- Panic tests will catch silent message changes immediately; a refactor that alters the panic string now fails rather than passing silently.
- Subtest-per-image isolation means a single failing image in `TestGetContainerPin_DefaultMCPImagesArePinned` no longer aborts the rest.
- Table-driven consolidation of `resolveExactHardcodedPin` variants reduces ~40 lines of duplication and makes adding new edge-case rows trivial.
- `assert.Equal` replacing a per-key loop in the `initWarnings` subtest produces a single structured diff on failure rather than multiple repeating assertion messages.

#### Negative
- `assert.PanicsWithValue` couples tests to the exact panic message string; any future change to a panic message must also update the corresponding test assertion.
- Consolidating four separate named functions into one table-driven test reduces the ability to run a single targeted case by function name with `-run`; callers must now use `-run TestResolveExactHardcodedPin/case-name`.

#### Neutral
- The `countingResolver` helper type moves position in the file but its behavior is unchanged.
- Comment banners (`// --- resolveExactHardcodedPin ---`) are cosmetic and have no runtime effect.
- `TestGetActionPins_CacheCorrectnessOnRepeatedCalls` is a net-new test; it adds runtime but does not change existing behavior.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
