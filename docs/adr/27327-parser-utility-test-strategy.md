# ADR-27327: Test Strategy for Parser Package Utility Functions

**Date**: 2026-04-20
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `pkg/parser` package contains several utility functions that underpin frontmatter processing: `IsWorkflowSpec`, `isNotFoundError`, `frontmatterContainsExpressions`, and `processImportsFromFrontmatter`. These functions handle critical path logic for workflow file parsing, error classification, and import processing. Prior to this change, test coverage for these utilities was incomplete: existing tests called unexported helpers (e.g., `isWorkflowSpec`) rather than the exported API, error-path branches for `processImportsFromFrontmatter` were untested, and no dedicated coverage existed for `isNotFoundError` or `frontmatterContainsExpressions`. A test quality review surfaced these gaps and motivated a systematic coverage improvement targeting both correctness and long-term maintainability.

### Decision

We will test parser utility functions through their exported API surface (e.g., `IsWorkflowSpec` rather than the unexported `isWorkflowSpec`) and will provide comprehensive table-driven tests that cover empty inputs, truthy and falsy edge cases, and near-miss inputs for each utility predicate. For `processImportsFromFrontmatter`, we will explicitly test the engine-metadata propagation path and the missing-file error path, asserting on a specific error message substring via `require.ErrorContains`. Test function names within the package must be unique to avoid silent shadowing across files.

### Alternatives Considered

#### Alternative 1: Test Private Helpers Directly

Unexported functions such as `isWorkflowSpec` are accessible within the same package and could be tested directly without exporting them. This was the prior approach. It was rejected because it couples tests to internal naming; if the function is renamed, moved, or inlined, tests break without any change to observable public behavior. Testing through the exported API (`IsWorkflowSpec`) means tests survive internal refactoring and document the intended public contract.

#### Alternative 2: Integration-Level Tests Only

Parser utilities could be tested exclusively at the workflow-loading integration level — parsing a complete workflow file and asserting the final result. This was not chosen because integration tests are slower to execute, harder to debug on failure, and do not isolate which utility function is at fault. Focused unit-level table-driven tests provide faster feedback and pinpoint failures precisely.

### Consequences

#### Positive
- Tests are decoupled from private implementation details and survive internal refactoring without modification.
- Table-driven test structure makes adding new edge-case inputs trivial without writing new test functions.
- `require.ErrorContains` assertions on error paths produce actionable failure messages that name the expected error substring.
- Removing the duplicate `TestIsNotFoundError` name across test files prevents silent test shadowing detectable only by the Go toolchain.

#### Negative
- Functions such as `isNotFoundError` and `frontmatterContainsExpressions` remain unexported; tests in `pkg/parser` that exercise them cannot be moved to a black-box `parser_test` package without promoting those functions to exported, creating mild coupling to the package's internal API.
- The engine-content fixture file written to `tempDir` in `TestProcessImportsFromFrontmatter` adds setup complexity; future test cases in the same function must account for that fixture's presence.

#### Neutral
- The renamed function `TestIsNotFoundError_RemoteNested` in `import_remote_nested_test.go` disambiguates the two test functions but changes the test name string reported in CI output and `go test -v` listings.
- No production code is modified by this decision; all changes are confined to `_test.go` files.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### API Surface Testing

1. Tests for functions with both an exported and an unexported variant **MUST** call the exported variant (e.g., `IsWorkflowSpec` rather than `isWorkflowSpec`).
2. Tests **MUST NOT** call unexported functions when an exported equivalent exists or when the behavior is reachable through a higher-level exported function.
3. When an unexported function is promoted to exported, tests **SHOULD** be updated to call the exported variant in the same PR that makes the promotion.

### Table-Driven Test Structure

1. Tests for utility predicates and classifiers (e.g., `isNotFoundError`, `frontmatterContainsExpressions`) **MUST** use Go table-driven tests (`[]struct{ ... }` with `t.Run`).
2. Each table **MUST** include at minimum: an empty or zero-value input, at least one input that produces a `true`/non-nil result, and at least one near-miss input that is superficially similar to a truthy case but should produce a `false`/nil result.
3. Error-path test cases **SHOULD** assert on a specific error message substring using `require.ErrorContains` rather than asserting only that a non-nil error was returned.

### Test Function Naming

1. Test function names within a Go package **MUST** be globally unique across all `_test.go` files in that package.
2. When a naming collision is resolved by renaming a test function, the new name **SHOULD** include a suffix that identifies the originating file or scenario domain (e.g., `_RemoteNested`).

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
