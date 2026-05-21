# ADR-33834: Add file-close-not-deferred Linter

**Date**: 2026-05-21
**Status**: Draft
**Deciders**: Unknown

---

## Part 1 — Narrative (Human-Friendly)

### Context

An automated scan of the codebase (linter-miner run #12) found 3 instances where files opened via `os.Open` / `os.OpenFile` are closed manually instead of with `defer file.Close()` — `pkg/cli/workflows.go:307`, `pkg/cli/workflows.go:396`, and `pkg/cli/mcp_logs_guardrail.go:87`. Manual close is the canonical Go anti-pattern for file handles: early returns, panics, or new branches added later between open and close cause descriptor leaks, and forgetting `defer` is easy to miss in review. The repository already houses a family of small, focused, in-house analyzers under `pkg/linters/` (e.g., `largefunc`, `osexitinlibrary`, `regexpcompileinfunction`) registered through `cmd/linters/main.go`, so the established convention is to add another analyzer rather than rely on review discipline or external tooling.

### Decision

We will add a new static-analysis linter, `fileclosenotdeferred`, that flags functions where a file opened by `os.Open`, `os.Create`, or `os.OpenFile` is closed via a non-deferred `Close()` call. The linter lives under `pkg/linters/fileclosenotdeferred/`, is registered in `cmd/linters/main.go` alongside the existing analyzers, walks each `*ast.FuncDecl` body (excluding nested `*ast.FuncLit` closures to avoid false positives) tracking per-variable state keyed by `types.Object` (to correctly handle variable shadowing), and reports the open position when a variable has a manual `Close()` and no matching `defer`. Test files are excluded via the shared `pkg/linters/internal/filecheck.IsTestFile` helper.

### Alternatives Considered

#### Alternative 1: Fix the 3 known instances and rely on review

Patch the three flagged call sites to use `defer file.Close()` and trust reviewers to catch new instances. Rejected because the same anti-pattern was already present in three independent files, indicating that human review alone has not been sufficient; a mechanical check on every PR is cheaper and cannot be forgotten as the codebase grows.

#### Alternative 2: Use a third-party linter (e.g., `gocritic`, `bodyclose`)

General-purpose Go linters offer resource-leak rules with broader coverage. Rejected to stay consistent with the project's convention of small, focused, in-house analyzers under `pkg/linters/`, each as its own package with custom logic. Pulling in an external linter for a single rule introduces a new dependency surface, inconsistent rule configuration, and noise from rules the project has not opted into.

#### Alternative 3: Combine with a broader "resource hygiene" linter

Bundle this rule with future checks (e.g., `http.Response.Body` close, `sql.Rows` close, `io.Closer` in general) into one analyzer. Rejected because the existing convention is one analyzer per rule, which keeps each rule's `Analyzer.Doc` URL narrow, independently disable-able, and easy to extend without coupling unrelated checks.

### Consequences

#### Positive
- New non-deferred `Close()` patterns introduced after merge are caught by `make golint-custom` and fail in CI rather than landing on `main`.
- The linter follows the same `pkg/linters/<name>/` layout, `Analyzer` shape, and `testdata` convention as the sibling analyzers, so contributors can extend it without learning a new pattern.
- Creates incentive to clean up the 3 pre-existing manual-close sites in `pkg/cli/` to maintain a clean linter signal.

#### Negative
- Detection is structural and intentionally narrow: it matches only direct calls to `os.Open` / `os.Create` / `os.OpenFile` and direct `<var>.Close()` calls. It will miss aliased imports (`import o "os"`), files returned from helper functions, and `Close()` called on a field or selector. False negatives are accepted in exchange for zero false positives.
- The 3 pre-existing violations are not fixed in this PR, so the linter cannot be made a blocking CI gate without follow-up work or suppression.
- Adds one more analyzer to the registry, marginally increasing `cmd/linters/main.go` compile and run time.

#### Neutral
- Test files are deliberately excluded via `filecheck.IsTestFile`, matching the convention used by the sibling linters. Test fixtures may legitimately close files inline.
- The `hasManuaClose` typo has been corrected to `hasManualClose`; the state struct is internal only.
- The diagnostic reports at the open position, not the manual-close position, so the warning points the reader to the place where `defer` should be inserted.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Linter Behaviour

1. The analyzer **MUST** be exported as `fileclosenotdeferred.Analyzer` with `Name` equal to `"fileclosenotdeferred"`.
2. The analyzer **MUST** report a diagnostic for every local variable assigned from a call to `os.Open`, `os.Create`, or `os.OpenFile` when that variable's `Close()` method is invoked without a matching `defer` statement in the same `*ast.FuncDecl` body.
3. The analyzer **MUST NOT** report a diagnostic when the file variable's `Close()` is invoked via a `defer` statement anywhere in the same function body.
4. The analyzer **MUST NOT** report a diagnostic when the result of the open call is discarded with the blank identifier (`_`).
5. The analyzer **MUST NOT** report a diagnostic when the containing file is a Go test file as determined by `pkg/linters/internal/filecheck.IsTestFile`.
6. The diagnostic `Pos` **MUST** be the position of the originating open call (`os.Open` / `os.Create` / `os.OpenFile`), not the position of the manual `Close()`.
7. The diagnostic `Message` **SHOULD** read `"file Close() should be deferred immediately after successful open to prevent resource leaks"` so downstream tooling can match on a stable string.
8. The analyzer **MUST** declare `inspect.Analyzer` in its `Requires` list.

### Registration

1. The analyzer **MUST** be registered in `cmd/linters/main.go` via the `multichecker.Main` argument list alongside the existing custom analyzers.
2. The package import in `cmd/linters/main.go` **MUST** use the path `github.com/github/gh-aw/pkg/linters/fileclosenotdeferred`.

### Package Layout

1. The linter source **MUST** live under `pkg/linters/fileclosenotdeferred/`.
2. Test fixtures **MUST** live under `pkg/linters/fileclosenotdeferred/testdata/src/fileclosenotdeferred/` and **MUST** use `// want` comments compatible with `golang.org/x/tools/go/analysis/analysistest`.
3. The test fixtures **MUST** include at least one positive case (manual `Close()` flagged), one negative case (`defer file.Close()` not flagged), and one blank-identifier case (`_, err := os.Open(...)` not flagged).

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26245378772) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
