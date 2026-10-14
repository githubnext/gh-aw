# ADR-68780: Add a dedicated `deferred-error-unchecked` AST linter for discarded deferred errors

**Date**: 2026-10-14
**Status**: Draft
**Deciders**: Unknown (PR authored by the `linter-miner` agentic workflow)

---

### Context

The gh-aw codebase defers many cleanup calls (`defer os.Remove(...)`, `defer conn.Close()`, `defer f.Close()`) whose `error` return value is silently discarded. A code-pattern scan found recurring instances (for example `pkg/cli/edit_command.go:152`) where a cleanup failure — an orphaned temporary file, a leaked connection, an unflushed writer — would never surface. The repository already owns a family of in-house `golang.org/x/tools/go/analysis` linters under `pkg/linters/`, each registered in `pkg/linters/registry.go` and sharing helpers in `pkg/linters/internal/` (`analyzerutil`, `filecheck`, `nolint`). Two existing linters partially cover this space: `uncheckederror` (unchecked returns in normal statements) and `unchecked-deferred-close` (specifically `Close()` methods). Neither flags an arbitrary deferred call whose signature ends in `error`.

### Decision

We will add a new standalone analyzer package `pkg/linters/deferred-error-unchecked` (Go package `deferrerrorunchecked`) that visits every `*ast.DeferStmt`, resolves the callee signature via `pass.TypesInfo` (identifier, selector/method value, or the call expression's own type), and reports a diagnostic when the last result type is the builtin `error`. The analyzer reuses the shared `analyzerutil.New`/`analyzerutil.Preorder` scaffolding and honours the existing generated-file skip (`filecheck.ShouldSkipFilename`) and `nolint` directive index, and is registered alphabetically in `pkg/linters/registry.go`. The primary driver is detection coverage with zero disruption: a new, separately-named analyzer keeps the existing linters' diagnostics and suppression directives unchanged while extending enforcement to all deferred error-returning calls.

### Alternatives Considered

#### Alternative 1: Extend the existing `unchecked-deferred-close` linter

`unchecked-deferred-close` already walks deferred statements, so broadening its signature check from `Close()` to "any call returning `error`" would avoid a new package and a second AST traversal. It was rejected because it silently changes the meaning of an existing linter name and of every `//nolint` directive already written against it, and it would make the narrower, higher-confidence `Close()` rule impossible to enable or suppress independently.

#### Alternative 2: Adopt an off-the-shelf linter (`errcheck` with `check-blank`/defer support)

`errcheck` and `golangci-lint`'s error-check family already detect ignored errors including deferred ones, so no new code would be needed. It was rejected because gh-aw's linter suite is deliberately self-hosted in `pkg/linters/` with shared generated-file and `nolint` handling, and introducing an external analyzer here would bypass those conventions, add a dependency and configuration surface, and produce diagnostics that do not participate in the repo's existing suppression mechanism. *[TODO: verify whether `errcheck` is already available in the repository's lint toolchain.]*

### Consequences

#### Positive
- Cleanup failures in deferred calls (temp-file removal, connection close, flush) become visible at build/lint time rather than silently lost.
- The rule is independently toggleable and suppressible (`//nolint:deferrerrorunchecked`) without weakening or renaming existing linters.
- Implementation follows the established `pkg/linters/` pattern, so it inherits generated-file skipping, `nolint` support, and the standard `analysistest` fixture layout.

#### Negative
- Overlapping coverage with `unchecked-deferred-close` and `uncheckederror` means some call sites will now be reported by more than one linter, producing duplicate diagnostics until the rules are reconciled.
- The rule has no allow-list, so idiomatic and intentionally-ignored cleanups (for example `defer resp.Body.Close()`) will be flagged, likely requiring a broad wave of `//nolint` annotations or code churn across the repository.
- Signature resolution is limited to identifiers, method values, and directly typed call expressions; interface method expressions, function values held in struct fields, and some generic instantiations may be missed (false negatives).

#### Neutral
- Adds one more `*ast.DeferStmt` traversal to the analysis pass set, a small but non-zero lint-time cost.
- Test coverage is fixture-based under `testdata/src/deferred_error_unchecked/` (`bad.go` / good cases), matching the convention of the other linters in this package tree.
- The package directory (`deferred-error-unchecked`) and the Go package name (`deferrerrorunchecked`) intentionally differ, consistent with sibling linters.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
