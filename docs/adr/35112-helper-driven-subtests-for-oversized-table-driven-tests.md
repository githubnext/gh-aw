# ADR-35112: Helper-Driven Subtests for Oversized Table-Driven Tests

**Date**: 2026-05-27
**Status**: Draft
**Deciders**: Unknown (inferred from PR #35112)

---

## Part 1 — Narrative (Human-Friendly)

### Context

LintMonster's `largefunc` rule (max 60 lines) flagged several `*_test.go` functions across the
codebase whose length came almost entirely from inline test tables and repeated per-case
setup/assertion blocks — not from genuinely complex logic. Representative failures lived in
`pkg/agentdrain/anomaly_test.go`, `pkg/workflow/schedule_preprocessing_test.go`, and
`pkg/cli/access_log_test.go`. The team needs a uniform way to bring these tests under the
limit without changing test intent, splitting tests across unrelated files, or sweeping the
lint rule under the rug. Future contributors fixing additional `largefunc` violations need a
single recognizable pattern to follow, so refactors stay consistent across packages.

### Decision

We will refactor oversized table-driven `Test*` functions by **(1) promoting the inline
test-case slice to a package-level typed fixture** (`type fooTestCase struct { … }` and
`var fooTestCases = []fooTestCase{ … }`), **(2) keeping the top-level `Test*` function as a
thin subtest dispatcher** that iterates the fixture with `t.Run`, and **(3) moving repeated
setup and assertion logic into `t.Helper()`-marked functions** named
`run<Action>Case` / `assert<Action>Case`. Production code is never touched; behavior and
assertions are preserved exactly.

### Alternatives Considered

#### Alternative 1: Suppress the `largefunc` lint rule for `*_test.go`

Disable or relax LintMonster's length rule in test files only. Rejected because the rule
already catches real readability problems in tests — the very fact that the flagged
functions had repeated setup/assertion blocks indicates they were ripe for extraction. A
blanket suppression would hide future regressions and provide no pressure toward consistent
table-driven patterns.

#### Alternative 2: Split each oversized `Test*` into multiple `Test*` functions by case group

Break a single table-driven test into several siblings (`TestAnalyze_NewTemplate`,
`TestAnalyze_LowSimilarity`, …). Rejected because it fragments closely related cases,
duplicates setup boilerplate at the function level, and loses the at-a-glance table view of
all input/expectation pairs. It also obscures the relationship between cases that share a
single underlying behavior (e.g., boundary-condition cases for the same function).

#### Alternative 3: Keep tables inline but extract only the assertion helper

A minimal refactor that leaves the `[]struct{ … }{ … }` literal inside the `Test*` function
and only pulls out the assertion body. Rejected because the literal itself accounts for the
bulk of the lint-flagged line count; trimming the assertion alone is not enough to bring
the largest cases under 60 lines, and it produces an inconsistent pattern across files
(some files extract both, others only one).

### Consequences

#### Positive
- `Test*` dispatchers stay well under the 60-line ceiling, satisfying the `largefunc` rule
  without per-file lint suppressions.
- Test-case fixtures become reusable within the package — other tests can iterate the same
  slice for orthogonal checks without copy-pasting cases.
- Helper functions marked with `t.Helper()` produce accurate failure line numbers pointing
  at the caller's case, preserving test debuggability.
- Refactor is mechanical and pattern-driven, so subsequent `largefunc` test-file fixes can
  follow the same template with low cognitive overhead.

#### Negative
- Adds package-level identifiers (`<name>TestCase` types and `<name>TestCases` slices) that
  expand the test-package namespace; risk of name collisions grows as more files adopt the
  pattern.
- Introduces one level of indirection — a reader must jump from the `Test*` body to the
  assertion helper and (in some cases) to a second helper (e.g., `assertScheduleAndCron`)
  to follow the full assertion logic.
- The convention is enforced only by reviewer discipline; nothing prevents a future
  contributor from re-inlining cases or naming helpers inconsistently.

#### Neutral
- Only representative locations were refactored in this PR; the rest of the `largefunc`
  test-file backlog remains and will be addressed in follow-up changes using this same
  template.
- Package-level fixture slices are mutable in Go. Tests that need to mutate per-case
  fields must copy the case (`tt := tt` or value-receiver helpers) — the current code
  passes by value, so this is observed in practice but not statically enforced.
- The pattern is specific to **table-driven tests**; tests that are oversized for other
  reasons (long setup, multi-step orchestration) need a different remediation.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**,
> **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be
> interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Test-Case Fixtures

1. When refactoring an oversized table-driven test to satisfy the `largefunc` rule,
   implementations **MUST** lift the inline `[]struct{ … }{ … }` literal to a package-level
   `var <name>TestCases = []<name>TestCase{ … }` declaration with an accompanying named
   `type <name>TestCase struct { … }`.
2. The fixture type and fixture slice **MUST** be declared in the same `*_test.go` file as
   the `Test*` function that consumes them.
3. The fixture type **MUST NOT** export fields beyond what the consuming `Test*` and helper
   functions require; speculative fields **MUST NOT** be added.
4. Implementations **SHOULD** name the type and slice after the function under test
   (e.g., `parseSquidLogLineTestCase` / `parseSquidLogLineTestCases`) so the relationship is
   obvious from the identifier alone.

### Test Dispatcher Shape

1. The top-level `Test*` function **MUST** be a thin loop: it **MUST** iterate the
   package-level fixture, call `t.Run(tt.name, …)`, and delegate the body to a helper.
2. The top-level `Test*` function **MUST NOT** contain setup logic, assertion logic, or
   inline test-case literals once refactored under this ADR.
3. The dispatcher loop body **SHOULD** be no more than a `t.Run` call invoking a single
   helper; additional logic in the dispatcher **SHOULD** be moved into the helper instead.

### Assertion and Setup Helpers

1. Per-case execution and assertion logic **MUST** be extracted into one or more
   helper functions named with the prefix `run<Action>Case` (when the helper also performs
   setup) or `assert<Action>Case` (when the helper performs only assertions).
2. Every extracted helper **MUST** call `t.Helper()` as its first statement so failure
   reports point at the caller's case.
3. Helpers **MUST NOT** alter the assertion semantics of the original test; the set of
   assertions and their failure messages **MUST** be preserved.
4. When an assertion block is reused by multiple cases or multiple `Test*` functions,
   implementations **SHOULD** further factor it into a secondary helper (as
   `assertScheduleAndCron` does in `pkg/workflow/schedule_preprocessing_test.go`).

### Scope and Non-Goals

1. This ADR **MUST NOT** be applied as a justification to modify production
   (non-`_test.go`) code.
2. This ADR **MUST NOT** be applied to tests that are not table-driven; such tests
   **SHOULD** be remediated by extracting setup/orchestration helpers instead, on a
   case-by-case basis.
3. Refactors under this ADR **SHOULD** be applied incrementally — the PR introducing this
   ADR refactors only representative locations, and follow-up PRs **MAY** extend the
   pattern to additional flagged files.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and
**MUST NOT** requirements above. A `*_test.go` file refactored to address a `largefunc`
violation in a table-driven `Test*` function is non-conformant if any of the following
hold: the test-case slice remains inline, the dispatcher contains setup or assertion logic,
extracted helpers omit `t.Helper()`, or production code was modified as part of the
refactor.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
