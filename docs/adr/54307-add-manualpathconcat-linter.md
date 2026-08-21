# ADR-54307: Add manualpathconcat Linter to Flag Manual "/" Path Concatenation

**Date**: 2026-08-20
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

Manual `"/"` separator path concatenation (`dir + "/" + file`) appears in 50+ places across `pkg/` and `cmd/` in gh-aw. This pattern is error-prone: it can produce double or missing separators when an operand already ends with `/`, it bypasses the `Clean`-style normalization that `filepath.Join` performs, and it hard-codes the POSIX forward-slash separator rather than the OS-specific one. No existing golangci-lint rule or custom gh-aw linter catches this pattern, leaving it undetected in code review and CI.

### Decision

We will add a new `go/analysis` analyzer, `manualpathconcat`, to `pkg/linters/` following the established layout of `stringsjoinone` and `largefunc`. The analyzer matches the `X + "/" + Y` AST pattern (parsed as `(X + "/") + Y`) on `*ast.BinaryExpr` and reports a single diagnostic per concatenation chain, recommending `filepath.Join` or `path.Join`. It skips generated files, `//nolint:manualpathconcat`-annotated lines, and constant-valued expressions (where `filepath.Join` is not valid Go). The analyzer is registered in `registry.go` and documented in `doc.go` and `README.md`, but intentionally **not** added to the enforced CI analyzer list in `.github/workflows/cgo.yml`, allowing the 51 pre-existing violations to be remediated incrementally.

### Alternatives Considered

#### Alternative 1: Rely on an existing golangci-lint rule (e.g., `gocritic` or `depguard`)

golangci-lint's `gocritic` checker and `depguard` can detect some bad patterns, but neither has a built-in rule that matches the `X + "/" + Y` concatenation shape specifically. Configuring `depguard` to catch string-literal `"/"` in binary expressions would require a regex-based approach that produces many false positives (e.g., URL construction, non-path separators) and would be difficult to tune per-site. The custom `go/analysis` approach gives precise AST-level matching with zero false positives on pure string concat shapes.

#### Alternative 2: Provide a `SuggestedFix`-enabled analyzer for automated rewrites

The analyzer could emit `analysis.SuggestedFix` edits to automatically rewrite `dir + "/" + file` to `filepath.Join(dir, file)`. This was rejected because the rewrite requires adding or verifying an existing `path/filepath` import in each file, and reordering operands in a chain (e.g., `a + "/" + b + "/" + c`) could change evaluation order for side-effecting expressions like `os.TempDir()`. A diagnostic-only approach is safer: it flags the problem and lets the developer apply the correct fix with full context.

### Consequences

#### Positive
- Catches all 50+ existing sites of unsafe manual path construction at analysis time, making them visible to developers and reviewers.
- Prevents future occurrences from being introduced undetected; the linter integrates with the existing gh-aw custom analyzer infrastructure and nolint suppression mechanism.

#### Negative
- The 51 pre-existing violations must be remediated manually (or suppressed per-line with `//nolint:manualpathconcat`) before the analyzer can be added to the enforced CI list; this is a deferred cleanup cost.
- No `SuggestedFix` is provided, so developers must manually apply `filepath.Join` rewrites and manage the import, increasing per-site remediation effort.

#### Neutral
- The analyzer is registered in `allAnalyzers` (available to tools that use the registry) but excluded from CI enforcement, meaning it runs in opt-in contexts only until the pre-existing violations are cleared.
- The diagnostic count in `doc.go` and `spec_test.go` advances from 66 to 67, keeping the registry/README sync tests in step.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
