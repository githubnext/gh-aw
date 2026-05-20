# ADR-33608: Add regexp-compile-in-function Linter

**Date**: 2026-05-20
**Status**: Draft
**Deciders**: Unknown

---

## Part 1 — Narrative (Human-Friendly)

### Context

A scan of `pkg/` revealed 20+ instances of `regexp.MustCompile` or `regexp.Compile` invoked inside function bodies (e.g., `pkg/stringutil/sanitize.go:143`), including hot paths used during sanitization, parsing, and CLI execution. Each call re-parses and re-optimizes the same pattern on every invocation, which is the canonical Go anti-pattern for regexp use. Existing in-house linters (`largefunc`, `osexitinlibrary`, `rawloginlib`, `ssljson`) already enforce repository-specific Go conventions via the `golang.org/x/tools/go/analysis` framework, but no linter covered this pattern. The repository goal of "catch the anti-pattern in code review, not after a perf regression" motivated adding another analyzer rather than relying on manual review.

### Decision

We will add a new static-analysis linter, `regexpcompileinfunction`, that flags calls to `regexp.MustCompile` and `regexp.Compile` occurring inside function bodies, methods, function literals, and loops, and recommends moving the compiled pattern to a package-level variable. The linter is implemented under `pkg/linters/regexpcompileinfunction/`, registered in `cmd/linters/main.go` alongside the existing analyzers, and uses `inspector.WithStack` AST traversal to determine function context. Test files are excluded via the existing `pkg/linters/internal/filecheck.IsTestFile` helper.

### Alternatives Considered

#### Alternative 1: Rely on code review and documentation

Document the anti-pattern in a CONTRIBUTING guide and trust reviewers to catch new instances. Rejected because the diff already shows ~20 pre-existing violations across `pkg/`, indicating that human review alone has not been sufficient. A linter that runs on every PR enforces the rule mechanically and cannot be forgotten.

#### Alternative 2: Use a third-party linter (e.g., `gocritic`, `revive`)

Some general-purpose Go linters offer regexp checks. Rejected to stay consistent with the project's pattern of small, focused, in-house analyzers under `pkg/linters/` — each existing analyzer (`largefunc`, `osexitinlibrary`, `rawloginlib`, `ssljson`) is its own package with custom logic. Adopting an external linter for a single rule would introduce a new dependency surface and inconsistent rule configuration.

#### Alternative 3: Build a single, broader "regexp hygiene" linter

Combine this rule with future regexp checks (e.g., unused capture groups, anchored-pattern hints) into a single analyzer. Rejected because the existing convention is one analyzer per rule, which keeps each rule's `Analyzer.Doc` URL and configuration narrow and independently disable-able.

### Consequences

#### Positive
- New regexp-in-function anti-patterns introduced after merge will be caught by `make golint-custom` and fail in CI rather than landing on `main`.
- The linter follows the existing `pkg/linters/<name>/` layout, so contributors familiar with `largefunc` or `rawloginlib` can read and extend it without learning a new pattern.
- Forces eventual cleanup of the ~20 pre-existing violations in `pkg/` to maintain a clean linter signal.

#### Negative
- The ~20 pre-existing violations are not addressed in this PR, so the linter cannot be made a blocking CI gate until those are fixed (or suppressed). Until then, the rule is advisory.
- The check uses a simple AST identifier match for `regexp.MustCompile` / `regexp.Compile`, which will miss aliased imports (`import re "regexp"`) and indirect calls. False negatives are accepted in exchange for zero false positives.
- Adds one more analyzer to the registry, marginally increasing `cmd/linters/main.go` compile and run time.

#### Neutral
- Test files are deliberately excluded via `filecheck.IsTestFile`, matching the convention used by the sibling linters. Test fixtures intentionally compile regexps inline and should not be flagged.
- Detection is structural (AST-based), so it does not require a build of the analyzed package and runs in the same pass as the other custom linters.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Linter Behaviour

1. The analyzer **MUST** be exported as `regexpcompileinfunction.Analyzer` with `Name` equal to `"regexpcompileinfunction"`.
2. The analyzer **MUST** report a diagnostic for every call expression whose function is `regexp.MustCompile` or `regexp.Compile` when that call is syntactically inside an `*ast.FuncDecl` or `*ast.FuncLit`.
3. The analyzer **MUST NOT** report a diagnostic for `regexp.MustCompile` or `regexp.Compile` calls that occur at package level (e.g., in a top-level `var` declaration).
4. The analyzer **MUST NOT** report a diagnostic when the containing file is a Go test file as determined by `pkg/linters/internal/filecheck.IsTestFile`.
5. The diagnostic `Message` **SHOULD** read `"regexp compilation inside function should be moved to package-level variable"` so downstream tooling can match on a stable string.
6. The analyzer **MUST** declare `inspect.Analyzer` in its `Requires` list and use `inspector.WithStack` (or an equivalent stack-aware traversal) to determine whether a call is inside a function.

### Registration

1. The analyzer **MUST** be registered in `cmd/linters/main.go` via `multichecker.Main` (or the equivalent entry point) alongside the existing custom analyzers.
2. The package import in `cmd/linters/main.go` **MUST** use the path `github.com/github/gh-aw/pkg/linters/regexpcompileinfunction`.

### Package Layout

1. The linter source **MUST** live under `pkg/linters/regexpcompileinfunction/`.
2. Test fixtures **MUST** live under `pkg/linters/regexpcompileinfunction/testdata/src/regexpcompileinfunction/` and **MUST** use `// want` comments compatible with `golang.org/x/tools/go/analysis/analysistest`.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26185856036) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
