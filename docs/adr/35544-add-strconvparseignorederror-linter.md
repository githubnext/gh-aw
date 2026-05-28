# ADR-35544: Add strconvparseignorederror Linter

**Date**: 2026-05-28
**Status**: Draft
**Deciders**: Unknown

---

## Part 1 — Narrative (Human-Friendly)

### Context

Issue #35389 reported a pattern of silent error discards from `strconv` parsing calls (`Atoi`, `ParseInt`, `ParseFloat`, `ParseBool`, `ParseUint`) at multiple production sites, including `pkg/parser/schedule_time_utils.go:72-73`, `pkg/parser/schedule_parser.go:151`, and `pkg/semverutil/semverutil.go:86,89,92`. When the error return of these functions is discarded with `_`, a parse failure returns the zero value of the result type (e.g., `0` for `Atoi`) and downstream logic silently consumes corrupted data — producing wrong schedule times, malformed semver components, and miscomputed configuration values without any visible failure signal. The repository already houses a family of small, focused, in-house analyzers under `pkg/linters/` (e.g., `uncheckedtypeassertion`, `fileclosenotdeferred`, `manualmutexunlock`, `fprintlnsprintf`, `regexpcompileinfunction`) registered through `cmd/linters/main.go`, so the established convention is to add another analyzer rather than rely on review discipline or an external tool.

### Decision

We will add a new static-analysis linter, `strconvparseignorederror`, that flags every two-LHS assignment of the form `_, _ := strconv.<Fn>(...)` where `<Fn>` is one of `Atoi`, `ParseInt`, `ParseFloat`, `ParseBool`, or `ParseUint` and the second LHS is the blank identifier. The linter lives under `pkg/linters/strconvparseignorederror/`, is registered in `cmd/linters/main.go` alongside the existing analyzers, walks each `*ast.AssignStmt` via the shared `inspect.Analyzer`, and verifies the call receiver actually resolves to the `strconv` package by consulting `pass.TypesInfo.Uses` rather than matching the identifier name `strconv` lexically (so that identically-named methods on other packages do not produce false positives). The implementation mirrors the structure of ADR-34738 (`uncheckedtypeassertion`), ADR-34498 (`fprintlnsprintf`), and ADR-34091 (`manualmutexunlock`) for consistency.

### Alternatives Considered

#### Alternative 1: Fix the known instances and rely on review

Patch the discarded errors flagged in issue #35389 (the `schedule_time_utils.go`, `schedule_parser.go`, and `semverutil.go` sites) and trust reviewers to catch new instances on future PRs. Rejected because the same shape appeared in multiple independent files across `pkg/`, indicating that human review has not been sufficient to catch it as code lands. A mechanical check on every PR is cheaper than reviewer attention and cannot regress.

#### Alternative 2: Use a general-purpose third-party linter (`errcheck`, `gocritic`)

`errcheck` flags every discarded error from every function in the codebase, and `gocritic` includes overlapping rules. Rejected because the project's convention is small, focused, in-house analyzers under `pkg/linters/`, each as its own package with custom logic. `errcheck` in particular would produce a large volume of diagnostics across the repository that the team has not opted into and that are unrelated to this specific high-signal pattern; opting in to a single rule from a broad linter is not how the existing analyzer registry is structured.

#### Alternative 3: Broaden the rule to any function returning `(T, error)`

Flag every two-LHS assignment whose RHS returns `(T, error)` and whose second LHS is `_`, regardless of which package the call targets. Rejected because the cost/benefit is highly uneven: `strconv` parse failures are particularly insidious because the zero return value is a plausible-looking number, whereas many other `(T, error)` returns produce a `nil` value or a sentinel that downstream code already checks. Starting narrowly at `strconv` matches the evidence in issue #35389 and avoids a flood of low-signal diagnostics; if a broader rule is later justified, it can be added as a separate analyzer.

### Consequences

#### Positive
- New `_, _ := strconv.<Fn>(...)` discards introduced after merge are caught by `make golint-custom` and fail in CI rather than landing on `main`, preventing future recurrences of the silent-corruption class described in issue #35389.
- The linter follows the same `pkg/linters/<name>/` layout, `Analyzer` shape, and `testdata` convention as sibling analyzers, so contributors can extend it without learning a new pattern.
- Receiver resolution via `TypesInfo.Uses` eliminates a class of false positives where another package exports a method named `Atoi`, `ParseInt`, etc., on a value also called `strconv`.

#### Negative
- The rule is intentionally narrow to `strconv` parsing calls and will not catch the same silent-corruption pattern when it arises from other parsers (e.g., `time.ParseDuration`, `net.ParseIP`, `url.Parse`). Each additional parser family requires either expanding the function set or a new analyzer.
- The pre-existing discards surfaced by the issue's pattern scan are not fixed in this PR, so the linter cannot be made a blocking CI gate without follow-up work or per-site suppression.
- Adds one more analyzer to the registry, marginally increasing `cmd/linters/main.go` compile and run time.

#### Neutral
- The detection is syntactic: it requires the literal `_` blank identifier as the second LHS. An assignment of the form `n, err := strconv.Atoi(s); _ = err` is not flagged because the error is captured first; this is intentional and matches the issue's scoped pattern.
- Test files are not specifically excluded by this analyzer, unlike some sibling linters that exclude tests via `filecheck.IsTestFile`. Whether to suppress this rule in tests is left to future iteration if test-file noise emerges in practice.
- The diagnostic is positional only — it does not emit a suggested-fix code action. Authors must rewrite the call manually.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Linter Behaviour

1. The analyzer **MUST** be exported as `strconvparseignorederror.Analyzer` with `Name` equal to `"strconvparseignorederror"`.
2. The analyzer **MUST** report a diagnostic for every `*ast.AssignStmt` that satisfies **all** of the following:
   1. `len(Lhs) == 2` and `len(Rhs) == 1`.
   2. The second element of `Lhs` is an `*ast.Ident` whose `Name` is `"_"` (the blank identifier).
   3. The single element of `Rhs` is an `*ast.CallExpr` whose `Fun` is an `*ast.SelectorExpr`.
   4. The `Sel.Name` of that selector is one of `"Atoi"`, `"ParseInt"`, `"ParseFloat"`, `"ParseBool"`, or `"ParseUint"`.
   5. The `X` operand of the selector is an `*ast.Ident` for which `pass.TypesInfo.Uses[ident]` is a `*types.PkgName` whose `Imported().Path()` is `"strconv"`.
3. The analyzer **MUST NOT** report a diagnostic when the selector function name is not in the set listed in requirement 2.4, even if the error return is discarded.
4. The analyzer **MUST NOT** report a diagnostic when the call receiver resolves to a package other than `"strconv"`, even if the selector name matches one of the listed function names.
5. The analyzer **MUST NOT** report a diagnostic when the assignment captures the error in a non-blank identifier (e.g., `n, err := strconv.Atoi(s)`), even if `err` is subsequently discarded.
6. The diagnostic `Pos`/`End` range **MUST** cover the entire `*ast.CallExpr` node (i.e., `pass.ReportRangef(call, ...)`).
7. The diagnostic `Message` **SHOULD** include both the discarded function name and a short explanation of the silent-failure risk, of the shape `"error return from strconv.<Fn> is discarded; parse failures produce zero values silently"`, so downstream tooling can match on a stable substring.
8. The analyzer **MUST** declare `inspect.Analyzer` in its `Requires` list.
9. If `pass.TypesInfo.Uses[ident]` returns `nil` (e.g., when the package is dot-imported, type information is incomplete, or the receiver is shadowed), the analyzer **MUST** silently skip the candidate node rather than emitting a diagnostic or panicking.

### Registration

1. The analyzer **MUST** be registered in `cmd/linters/main.go` via the `multichecker.Main` argument list alongside the existing custom analyzers.
2. The package import in `cmd/linters/main.go` **MUST** use the path `github.com/github/gh-aw/pkg/linters/strconvparseignorederror`.

### Package Layout

1. The linter source **MUST** live under `pkg/linters/strconvparseignorederror/`.
2. Test fixtures **MUST** live under `pkg/linters/strconvparseignorederror/testdata/src/strconvparseignorederror/` and **MUST** use `// want` comments compatible with `golang.org/x/tools/go/analysis/analysistest`.
3. The test fixtures **MUST** include at least one positive case for each of the five listed `strconv` functions (`Atoi`, `ParseInt`, `ParseFloat`, `ParseBool`, `ParseUint`).
4. The test fixtures **MUST** include at least one negative case where the error return is captured in a named identifier and checked.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26600068083) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
