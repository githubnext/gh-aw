# ADR-46633: Add timenowsub Custom Linter for time.Now().Sub(t) → time.Since(t)

**Date**: 2026-07-19
**Status**: Draft
**Deciders**: Unknown

---

### Context

The repository maintains a collection of custom Go static analysis linters (under `pkg/linters/`) enforced via `cmd/linters/main.go`. A static scan of `pkg/` and `cmd/` identified occurrences of the verbose `time.Now().Sub(t)` pattern, which has a direct idiomatic replacement: `time.Since(t)`. Go's standard library documents `time.Since(t)` as shorthand for `time.Now().Sub(t)` (see [Go issue #16351](https://github.com/golang/go/issues/16351)), and this simplification is consistently flagged in Go code reviews. The pattern produces zero false positives since every `time.Now().Sub(x)` call has an exact mechanical replacement.

### Decision

We will add a new custom `timenowsub` analyzer to `pkg/linters/timenowsub/` and register it in `cmd/linters/main.go`. The analyzer uses `golang.org/x/tools/go/analysis` to detect AST nodes matching `time.Now().Sub(<arg>)` and reports a diagnostic with a `SuggestedFix` rewriting them to `time.Since(<arg>)`. Generated files and files with `//nolint:timenowsub` directives are skipped.

### Alternatives Considered

#### Alternative 1: Enable gosimple (S1012) from golangci-lint / staticcheck

`gosimple` check S1012 already covers this exact pattern. Enabling it from the broader `golangci-lint` / `staticcheck` toolchain would address the issue without writing custom code.

This was not chosen because: the repository's custom linter framework provides consistent enforcement, nolint-directive handling, generated-file skipping, and test infrastructure that the generic `gosimple` integration does not supply out of the box. Relying on `gosimple` also requires maintaining the `golangci-lint` configuration and ensuring S1012 is not accidentally disabled, whereas a custom linter is unconditionally active.

#### Alternative 2: Rely on code review without automated enforcement

Reviewers could flag `time.Now().Sub(t)` patterns manually during PR review without any tooling.

This was not chosen because: manual code review is inconsistent and does not scale — patterns are missed, especially in large diffs. The zero-false-positive nature of this check makes automated enforcement strictly better than human review for this specific pattern.

### Consequences

#### Positive
- Zero false positives: every flagged `time.Now().Sub(x)` call has a safe mechanical replacement.
- Automatic fix: the `SuggestedFix` in the diagnostic allows `gopls` and `go fix`-style tools to apply the rewrite with no manual intervention.
- Unconditional enforcement: the linter is always active regardless of golangci-lint configuration changes.
- Consistent with existing linter patterns: follows the same structure as other custom linters in `pkg/linters/`.

#### Negative
- New package to maintain: adds `pkg/linters/timenowsub/` to the custom linter collection, which must be updated if internal shared utilities (e.g., `astutil`, `filecheck`, `nolint`) change their APIs.
- Slightly increases binary size of the `cmd/linters` tool.

#### Neutral
- The linter only fires on `time.Now().Sub(x)` where the receiver is verified via type-checker to be `time.Now` — other `.Sub()` calls (e.g., `a.Sub(b)`) are unaffected.
- Test coverage is provided via `analysistest.RunWithSuggestedFixes` and a golden file, following the established test pattern for this linter suite.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
