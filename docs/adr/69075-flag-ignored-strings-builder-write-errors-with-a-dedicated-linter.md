# ADR-69075: Flag ignored `strings.Builder` write errors with a dedicated custom linter

**Date**: 2026-02-10
**Status**: Draft
**Deciders**: Unknown (drafted from PR #69075 by the Design Decision Gate)

---

### Context

`gh-aw` maintains a family of in-repo Go analysis linters under `pkg/linters/`, each a self-contained subpackage exposing a single `*analysis.Analyzer` and registered centrally in `pkg/linters/registry.go` (82 analyzers before this change). The PR reports 10+ places in workflow-compilation code (for example `cache_memory.go`, `compiler_safe_outputs_job.go`) where `strings.Builder.WriteString` / `WriteByte` / `WriteRune` results are dropped, so a write failure during output generation would be silently swallowed. Standard Go vet checks and `errcheck`-style tooling available to the repo do not currently surface this pattern with a low enough false-positive rate for the repo's conventions, and the repo's spec tests (`pkg/linters/spec_test.go`) and package docs (`pkg/linters/doc.go`) require every analyzer to be documented and registered in lockstep. The new check must respect the shared infrastructure for generated-file skipping and `nolint` directives so it does not fire on machine-generated sources.

### Decision

We will add a new analyzer subpackage `pkg/linters/writestring-error-ignored` (analyzer name `writestringerrorignored`) that reports `strings.Builder` write calls whose error return is discarded, and register it alongside the existing analyzers in `registry.go`, `doc.go`, and `spec_test.go`. Detection is AST- plus type-driven: the analyzer inspects `*ast.ExprStmt` and `*ast.AssignStmt` nodes, confirms the receiver's type resolves to `strings.Builder` (dereferencing pointers) and that the method's final result is `error`, and only then reports. To keep the signal high, it flags bare calls and fully-blank assignments (`_ = b.WriteString(...)`, `_, _ = b.WriteString(...)`) but deliberately does not flag partial captures (`n, _ := ...`) or `defer`red calls, which are `*ast.DeferStmt` nodes and therefore outside the node filter. Skipping is delegated to the existing `filecheck` (generated files) and `nolint` helpers rather than reimplemented.

### Alternatives Considered

#### Alternative 1: Rely on a general-purpose unchecked-error linter (`errcheck` / `go vet`)

A configured off-the-shelf unchecked-error linter would cover `strings.Builder` writes without new code to maintain. It was rejected because such linters flag *every* ignored error return across the codebase, which in this repo would produce a large volume of diagnostics on patterns that are intentionally ignored, and tuning the exclusion list to isolate `strings.Builder` would be roughly as much configuration work as the focused analyzer, with less control over the `defer` and partial-capture exemptions encoded here.

#### Alternative 2: Name-only matching on `WriteString` / `WriteByte` / `WriteRune`

A purely syntactic check (match the selector name, skip type resolution) would be shorter and would not need `pass.TypesInfo`. It was rejected because the same method names appear on `bytes.Buffer`, `bufio.Writer`, and many project types with different error semantics, so name-only matching would misfire outside the intended scope. The implementation therefore pays the cost of type resolution to pin the receiver to `strings.Builder`.

#### Alternative 3: Fix the ~10 known call sites and add no linter

A one-off cleanup of the identified occurrences in `cache_memory.go` and `compiler_safe_outputs_job.go` would remove today's instances with zero new tooling. It was rejected because nothing would prevent regressions in the many string-building paths of the compiler; a close call, but the repo has already standardised on codifying such conventions as analyzers.

### Consequences

#### Positive
- Ignored `strings.Builder` write errors are caught mechanically by `make golint-custom` instead of relying on review, closing a class of silent output-generation failures.
- Type-based receiver matching keeps the check scoped to `strings.Builder`, so adoption cost on the existing codebase is low and false positives are rare.
- The analyzer reuses the shared `filecheck` and `nolint` helpers, so generated code and explicitly-suppressed sites behave consistently with the other 82 analyzers.

#### Negative
- The analyzer count becomes 83, and every addition must be mirrored in three places (`registry.go`, `doc.go`, `spec_test.go`), increasing the maintenance surface and the chance of drift.
- Because `strings.Builder` writes effectively never fail (the documented error is always `nil`), some diagnostics will be pure ceremony, pushing authors toward `_ = ...`-style boilerplate or `nolint` directives at sites with no real risk. [TODO: verify whether the repo prefers an autofix/suggested-fix here.]
- Deferred calls and partial captures are intentionally exempt, so a genuinely ignored error in `defer b.WriteString(...)` is not reported — a known blind spot.

#### Neutral
- The package directory uses a hyphenated name (`writestring-error-ignored`) while the Go package identifier is `writestringerrorignored`, requiring a named import in the registry and spec test, consistent with other hyphenated analyzers in this tree.
- Existing violations in `cache_memory.go` and `compiler_safe_outputs_job.go` are not fixed by this PR; enabling the linter implies a follow-up cleanup or suppression pass.
- Test coverage is fixture-driven via `analysistest` against `testdata/src/writestring-error-ignored`, matching the convention used by sibling analyzers.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
