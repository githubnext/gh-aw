# ADR-34738: Add uncheckedtypeassertion Linter

**Date**: 2026-05-25
**Status**: Draft
**Deciders**: Unknown

---

## Part 1 — Narrative (Human-Friendly)

### Context

Issue #34580 reported a runtime panic in GraphQL response parsing caused by an unchecked single-value type assertion `project["id"].(string)` when the API returned a `nil` or unexpected-type value, while sibling fields in the same block used the safe two-value `s, ok := v.(string)` idiom. A pattern scan turned up additional single-value assertions across `pkg/` that follow the same risky shape. Go's single-value type assertion `x.(T)` is a latent panic whenever the asserted dynamic type does not hold, whereas the two-value form `v, ok := x.(T)` returns the zero value and `false` and is safe to recover from. The repository already houses a family of small, focused, in-house analyzers under `pkg/linters/` (e.g., `fileclosenotdeferred`, `manualmutexunlock`, `fprintlnsprintf`, `regexpcompileinfunction`) registered through `cmd/linters/main.go`, so the established convention is to add another analyzer rather than rely on review discipline or an external tool.

### Decision

We will add a new static-analysis linter, `uncheckedtypeassertion`, that flags every `*ast.TypeAssertExpr` whose result is consumed in a single-value position and recommends rewriting the call site as the safe two-value form `v, ok := x.(T)`. The linter lives under `pkg/linters/uncheckedtypeassertion/`, is registered in `cmd/linters/main.go` alongside the existing analyzers, walks each `*ast.TypeAssertExpr` via the shared `inspect.Analyzer`, builds a per-file parent map to decide whether the assertion sits on the RHS of an assignment with two LHS targets, and skips type-switch guards (`v.(type)`, identified by `TypeAssertExpr.Type == nil`) and test files (via `pkg/linters/internal/filecheck.IsTestFile`). The implementation mirrors the structure of ADR-34498 (`fprintlnsprintf`), ADR-34091 (`manualmutexunlock`), and ADR-33834 (`fileclosenotdeferred`) for consistency.

### Alternatives Considered

#### Alternative 1: Fix the known instances and rely on review

Patch the unchecked `project["id"].(string)` call site flagged in issue #34580 and the additional sites surfaced by the pattern scan, then trust reviewers to catch new instances. Rejected because the same shape appeared in multiple independent sites across `pkg/`, indicating that human review has not been sufficient to catch it. A mechanical check on every PR is cheaper than reviewer attention and cannot regress as new code lands.

#### Alternative 2: Use a third-party linter (e.g., `gocritic`, `forcetypeassert`)

`forcetypeassert` (and overlapping checks in `gocritic`, `staticcheck`) detect single-value type assertions. Rejected to stay consistent with the project's convention of small, focused, in-house analyzers under `pkg/linters/`, each as its own package with custom logic. Pulling in an external linter for a single rule introduces a new dependency surface, inconsistent rule configuration, and noise from rules the project has not opted into.

#### Alternative 3: Type-based filtering via `go/types`

Use `pass.TypesInfo` to suppress the diagnostic when the asserted type is itself an `interface{...}` that the operand already satisfies, or when the operand is statically known to be of the asserted type. Rejected because the analyzer is intentionally syntactic and structural: any single-value type assertion is a panic risk in production code regardless of static narrowing, and a uniform rule is easier to enforce than a context-sensitive one. False positives can be silenced at the call site by switching to the two-value form, which is always a safe rewrite.

### Consequences

#### Positive
- New `x.(T)` single-value type assertions introduced after merge are caught by `make golint-custom` and fail in CI rather than landing on `main`, preventing future recurrences of the issue #34580 panic class.
- The linter follows the same `pkg/linters/<name>/` layout, `Analyzer` shape, and `testdata` convention as sibling analyzers, so contributors can extend it without learning a new pattern.
- Creates incentive to clean up the pre-existing single-value assertion sites surfaced by the pattern scan to maintain a clean linter signal.

#### Negative
- Detection is structural: it flags every single-value type assertion outside test files, including cases where the author has consciously decided that a panic is acceptable (e.g., assertions on values the caller has already type-checked through an unrelated path). False positives must be silenced by rewriting to the two-value form `v, _ := x.(T)`, even when the discard `_` is the only practical handling.
- The pre-existing single-value assertion sites are not fixed in this PR, so the linter cannot be made a blocking CI gate without follow-up work or suppression.
- Adds one more analyzer to the registry, marginally increasing `cmd/linters/main.go` compile and run time, plus a per-file parent-map construction pass which is `O(n)` in AST node count per file.

#### Neutral
- Test files are deliberately excluded via `filecheck.IsTestFile`, matching the convention used by sibling linters. Tests may legitimately use single-value assertions on fixture data where a panic is the desired failure mode.
- Type-switch guards (`switch v.(type)`) are not flagged because the analyzer skips `TypeAssertExpr` nodes with `Type == nil`. This matches Go's own definition: a type-switch guard is not a runtime-panicking assertion.
- The diagnostic is positional only — it does not emit a suggested-fix code action. Authors must rewrite the call manually.
- The parent map is constructed per `*ast.File` rather than reused across analyzers, because the existing `inspect.Analyzer` does not expose parent links. The duplication is local to this analyzer and acceptable for the analyzer's size.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Linter Behaviour

1. The analyzer **MUST** be exported as `uncheckedtypeassertion.Analyzer` with `Name` equal to `"uncheckedtypeassertion"`.
2. The analyzer **MUST** report a diagnostic for every `*ast.TypeAssertExpr` whose `Type` field is non-nil and whose direct parent in the containing file's AST is **not** an `*ast.AssignStmt` with exactly two LHS expressions and exactly one RHS expression.
3. The analyzer **MUST NOT** report a diagnostic for any `*ast.TypeAssertExpr` whose `Type` field is `nil` (i.e., a type-switch guard `v.(type)`).
4. The analyzer **MUST NOT** report a diagnostic when the parent `*ast.AssignStmt` has `len(Lhs) == 2 && len(Rhs) == 1`, regardless of whether the second LHS is the blank identifier `_` or a named `ok` variable.
5. The analyzer **MUST NOT** report a diagnostic when the containing file is a Go test file as determined by `pkg/linters/internal/filecheck.IsTestFile`.
6. The diagnostic `Pos`/`End` range **MUST** cover the entire `*ast.TypeAssertExpr` node (i.e., `pass.ReportRangef(typeAssert, ...)`).
7. The diagnostic `Message` **SHOULD** include both the rendered asserted type and a recommendation to use the two-value form, of the shape `"type assertion x.(T) is unchecked and may panic; use the two-value form v, ok := x.(T) instead"`, so downstream tooling can match on a stable substring.
8. The analyzer **MUST** declare `inspect.Analyzer` in its `Requires` list.
9. The analyzer **MAY** rely on `pass.TypesInfo.TypeOf(typeAssert.Type)` to render the asserted type in the diagnostic; if the resolved type is `nil` the analyzer **MUST** silently skip emitting the diagnostic for that node rather than panicking or emitting a malformed message.

### Parent Resolution

1. The analyzer **MUST** construct a per-file map from each AST node to its direct parent node before the inspect pass evaluates type-assertion nodes, so that the two-value-form detection in requirement 4 is decidable in `O(1)` per node.
2. The parent map **MUST** be scoped to the `*ast.File` that contains the inspected node; nodes from a different file in the same package **MUST NOT** be present in the same parent map.

### Registration

1. The analyzer **MUST** be registered in `cmd/linters/main.go` via the `multichecker.Main` argument list alongside the existing custom analyzers.
2. The package import in `cmd/linters/main.go` **MUST** use the path `github.com/github/gh-aw/pkg/linters/uncheckedtypeassertion`.

### Package Layout

1. The linter source **MUST** live under `pkg/linters/uncheckedtypeassertion/`.
2. Test fixtures **MUST** live under `pkg/linters/uncheckedtypeassertion/testdata/src/uncheckedtypeassertion/` and **MUST** use `// want` comments compatible with `golang.org/x/tools/go/analysis/analysistest`.
3. The test fixtures **MUST** include at least one positive case for each of: a single-value assertion used as a `return` expression, and a single-value assertion bound via `:=` to a single LHS.
4. The test fixtures **MUST** include at least one negative case for each of: the two-value `v, ok :=` form, the two-value `v, _ :=` form with blank `ok`, and a type-switch guard `v.(type)`.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26413798298) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
