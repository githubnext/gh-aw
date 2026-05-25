# ADR-34498: Add fprintlnsprintf Linter

**Date**: 2026-05-24
**Status**: Draft
**Deciders**: Unknown

---

## Part 1 — Narrative (Human-Friendly)

### Context

An automated codebase scan (linter-miner run #17) found 11+ occurrences of `fmt.Fprintln(w, fmt.Sprintf(format, args...))` across `pkg/cli/trial_support.go` (5), `pkg/cli/update_merge.go` (3), and `pkg/cli/compile_watch.go` (3). This pattern is doubly wasteful: `fmt.Sprintf` allocates an intermediate formatted string, and `fmt.Fprintln` then allocates again to append `\n` and write it. The idiomatic form is `fmt.Fprintf(w, format+"\n", args...)` (or to embed `\n` in the format string), which eliminates the intermediate allocation. The repository already houses a family of small, focused, in-house analyzers under `pkg/linters/` (e.g., `fileclosenotdeferred`, `manualmutexunlock`, `regexpcompileinfunction`) registered through `cmd/linters/main.go`, so the established convention is to add another analyzer rather than rely on review discipline or external tooling.

### Decision

We will add a new static-analysis linter, `fprintlnsprintf`, that flags `fmt.Fprintln(w, fmt.Sprintf(...))` call expressions and recommends rewriting them as `fmt.Fprintf(w, ...)`. The linter lives under `pkg/linters/fprintlnsprintf/`, is registered in `cmd/linters/main.go` alongside the existing analyzers, walks each `*ast.CallExpr` via the shared `inspect.Analyzer`, and reports a diagnostic when the outer call is `fmt.Fprintln` with at least two arguments whose final argument is a direct call to `fmt.Sprintf`. Test files are excluded via the shared `pkg/linters/internal/filecheck.IsTestFile` helper. The implementation mirrors the structure of ADR-34091 (`manualmutexunlock`) and ADR-33834 (`fileclosenotdeferred`) for consistency.

### Alternatives Considered

#### Alternative 1: Fix the 11+ known instances and rely on review

Patch each flagged call site to use `fmt.Fprintf` and trust reviewers to catch new instances. Rejected because the same pattern was present in 11+ independent sites across three files in `pkg/cli/`, indicating that human review has not been sufficient to catch it. A mechanical check on every PR is cheaper than reviewer attention and cannot regress as new code lands.

#### Alternative 2: Use a third-party linter (e.g., `gocritic`, `staticcheck`)

General-purpose Go linters offer overlapping checks for redundant `Sprintf` wrappers. Rejected to stay consistent with the project's convention of small, focused, in-house analyzers under `pkg/linters/`, each as its own package with custom logic. Pulling in an external linter for a single rule introduces a new dependency surface, inconsistent rule configuration, and noise from rules the project has not opted into.

#### Alternative 3: Bundle with a broader "redundant fmt wrapper" linter

Generalize the check to flag any `fmt.Print*(fmt.Sprintf(...))` family (`Println(Sprintf(...))`, `Fprintln(Sprintf(...))`, `Fprint(Sprintf(...))`, etc.) inside a single analyzer. Rejected because the existing convention is one analyzer per rule, which keeps each `Analyzer.Doc` URL narrow, independently disable-able, and easy to extend without coupling unrelated checks. Sibling patterns can be added later as separate analyzers if the evidence warrants it.

### Consequences

#### Positive
- New `fmt.Fprintln(w, fmt.Sprintf(...))` occurrences introduced after merge are caught by `make golint-custom` and fail in CI rather than landing on `main`.
- The linter follows the same `pkg/linters/<name>/` layout, `Analyzer` shape, and `testdata` convention as sibling analyzers, so contributors can extend it without learning a new pattern.
- Creates incentive to clean up the 11+ pre-existing call sites in `pkg/cli/trial_support.go`, `pkg/cli/update_merge.go`, and `pkg/cli/compile_watch.go` to maintain a clean linter signal.

#### Negative
- Detection is structural and intentionally narrow: it matches only direct calls where the outer function is `fmt.Fprintln` and the last argument is a direct `fmt.Sprintf` call. It will miss equivalent constructions routed through an intermediate variable (`s := fmt.Sprintf(...); fmt.Fprintln(w, s)`) or where either call is renamed at import time. False negatives are accepted in exchange for a near-zero false-positive rate.
- The 11+ pre-existing violations are not fixed in this PR, so the linter cannot be made a blocking CI gate without follow-up work or suppression.
- Adds one more analyzer to the registry, marginally increasing `cmd/linters/main.go` compile and run time.

#### Neutral
- Test files are deliberately excluded via `filecheck.IsTestFile`, matching the convention used by sibling linters. Tests may legitimately format strings inline for fixture readability.
- The analyzer relies on syntactic identification (`fmt.<Name>` `*ast.SelectorExpr` matching) rather than `go/types` resolution. This is consistent with sibling linters and keeps the analyzer cheap, but means a user-defined package aliased as `fmt` (or `fmt` aliased to another name) would not be matched.
- The diagnostic is positional only — it does not emit a suggested-fix code action. Authors must rewrite the call manually.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Linter Behaviour

1. The analyzer **MUST** be exported as `fprintlnsprintf.Analyzer` with `Name` equal to `"fprintlnsprintf"`.
2. The analyzer **MUST** report a diagnostic for every `*ast.CallExpr` whose function selector matches `fmt.Fprintln`, whose argument list has length `>= 2`, and whose final argument is itself a `*ast.CallExpr` with function selector matching `fmt.Sprintf`.
3. The analyzer **MUST NOT** report a diagnostic when the outer call has fewer than two arguments.
4. The analyzer **MUST NOT** report a diagnostic when the final argument is anything other than a direct `fmt.Sprintf` call expression (e.g., a variable reference, a different function, or a parenthesized expression).
5. The analyzer **MUST NOT** report a diagnostic when the containing file is a Go test file as determined by `pkg/linters/internal/filecheck.IsTestFile`.
6. The diagnostic `Pos` **MUST** be the position of the outer `fmt.Fprintln` call expression.
7. The diagnostic `Message` **SHOULD** read `"use fmt.Fprintf instead of fmt.Fprintln(w, fmt.Sprintf(...))"` so downstream tooling can match on a stable string.
8. The analyzer **MUST** declare `inspect.Analyzer` in its `Requires` list.
9. Selector matching **MUST** check that the receiver identifier is literally `fmt` and the method name is the expected `Fprintln` / `Sprintf`; the analyzer **MAY** rely on syntactic identification and is **NOT REQUIRED** to consult `go/types`.

### Registration

1. The analyzer **MUST** be registered in `cmd/linters/main.go` via the `multichecker.Main` argument list alongside the existing custom analyzers.
2. The package import in `cmd/linters/main.go` **MUST** use the path `github.com/github/gh-aw/pkg/linters/fprintlnsprintf`.

### Package Layout

1. The linter source **MUST** live under `pkg/linters/fprintlnsprintf/`.
2. Test fixtures **MUST** live under `pkg/linters/fprintlnsprintf/testdata/src/fprintlnsprintf/` and **MUST** use `// want` comments compatible with `golang.org/x/tools/go/analysis/analysistest`.
3. The test fixtures **MUST** include at least one positive case (`fmt.Fprintln(w, fmt.Sprintf(...))` flagged) and at least one negative case (plain `fmt.Fprintln` with a literal string, and `fmt.Fprintf` with embedded `\n` not flagged).

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26368672971) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
