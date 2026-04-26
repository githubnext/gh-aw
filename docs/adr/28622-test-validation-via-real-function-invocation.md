# ADR-28622: Test Validation Logic via Real Function Invocation Rather Than Inline Re-implementation

**Date**: 2026-04-26
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `pkg/cli/health_command_test.go` test suite contained a `TestHealthConfigValidation` function that duplicated the days-validation logic inline rather than exercising the real `RunHealth` function. Additionally, `TestHealthCommand` used `assert.NotNil` for flag object lookups, which would cause a confusing nil-pointer panic if the assertion failed instead of stopping the test cleanly. Coverage of the JSON output path of `displayDetailedHealth` and of edge-case days values (0, -1, 91, 365) was absent. These gaps meant tests could pass even when the production validation path was broken.

### Decision

We will test the health command's validation behavior by calling `RunHealth` directly rather than re-implementing the validation predicate inside the test. We will use `require.NotNil` (and `require.NoError`) for any nil- or error-check whose failure would make subsequent assertions meaningless or panicky. New table-driven tests will cover all boundary values of the `days` parameter and the JSON output path of `displayDetailedHealth`.

### Alternatives Considered

#### Alternative 1: Keep inline validation reimplementation in tests

The existing approach checked `tt.config.Days != 7 && tt.config.Days != 30 && tt.config.Days != 90` directly in the test body. This keeps tests independent of `RunHealth` internals, but it duplicates the production validation predicate and will silently pass even if `RunHealth` loses its validation entirely, providing false confidence.

#### Alternative 2: Introduce a mock or stub for RunHealth to isolate validation

A dedicated `ValidateDays(days int) error` function could be extracted and tested independently without invoking `RunHealth`. This provides tighter unit isolation, but it requires production-code refactoring beyond the scope of improving existing test quality and postpones the coverage gains indefinitely.

### Consequences

#### Positive
- Tests now exercise the actual `RunHealth` validation path; a regression in production validation will break tests immediately.
- `require.NotNil` halts the test on nil rather than causing a confusing panic-in-assert, making failures easier to diagnose.
- JSON output path of `displayDetailedHealth` gains explicit coverage, including the nil-runs and empty-runs boundary cases.

#### Negative
- Tests now depend on `RunHealth`'s exact error message format (`"invalid days value: %d"`, `"Must be 7, 30, or 90"`), coupling test assertions to production string literals that could change independently.
- Valid-days test cases cannot assert a clean nil error because `RunHealth` may return a GitHub API error in CI environments without credentials; tests must tolerate that path.

#### Neutral
- The `require` package is introduced as an additional import alongside `assert`; both remain from the same `testify` library.
- Test file line count increases significantly (37 deletions → 141 additions), which will register as a code-volume change in automated gates.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Test Implementation

1. Tests for `RunHealth` **MUST** invoke `RunHealth` directly rather than re-implementing its validation predicates inline.
2. Test assertions that dereference a pointer or use a value from a previous call **MUST** use `require.NotNil` or `require.NoError` (not `assert.NotNil` / `assert.NoError`) so the test halts before a nil-pointer panic can occur.
3. Tests for `days` validation **MUST** cover at least the following boundary values: `0`, any negative value, `15` (a plausible but invalid value), `91`, and `365`, in addition to the three valid values `7`, `30`, `90`.
4. Tests for `days` validation **MUST** assert that the returned error message contains both the invalid value (e.g., `"invalid days value: 0"`) and the set of valid options (e.g., `"Must be 7, 30, or 90"`).
5. Tests that exercise JSON output paths (e.g., `displayDetailedHealth` with `JSONOutput: true`) **MUST** capture stdout, unmarshal the output as the declared JSON type, and assert at minimum that the top-level named fields match the input configuration.
6. Tests **SHOULD NOT** assert a nil error for `RunHealth` calls with valid `days` values when the test environment may lack GitHub API credentials; instead they **SHOULD** assert only that the error (if any) does not contain `"invalid days value"`.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24966509091) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
