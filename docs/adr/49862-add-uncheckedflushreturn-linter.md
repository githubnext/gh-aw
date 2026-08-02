# ADR-49862: Add `uncheckedflushreturn` Linter to Flag Discarded Flush Errors

**Date**: 2026-08-02
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The repository's custom Go linter suite (`pkg/linters/`) enforces code quality rules beyond what `golangci-lint` covers. The `errcheck` linter is disabled repo-wide, which means `Flush()` method calls on buffered writers (e.g. `bufio.Writer`, `tabwriter.Writer`) can silently discard their error return value. When a flush fails, buffered data is silently dropped with no indication to the caller or user. The Linter Miner workflow identified two live violations of this pattern in `pkg/cli/logs_format_compact.go`, confirming that the gap was real and already causing undetected risk.

### Decision

We will add a new custom `go/analysis` analyzer named `uncheckedflushreturn` that flags `Flush()` method calls where the `error` return value is discarded — either as a bare expression statement (`tw.Flush()`) or via explicit blank assignment (`_ = tw.Flush()`). The analyzer uses type information to confirm the method signature is `func() error`, avoiding false positives on `Flush` methods that do not return errors. It is registered in `pkg/linters/registry.go` alongside all other custom analyzers and follows the same `//nolint:uncheckedflushreturn` suppression convention.

### Alternatives Considered

#### Alternative 1: Re-enable `errcheck` repo-wide

Re-enabling `errcheck` would catch all ignored error returns, including `Flush()`, without requiring a custom analyzer. However, `errcheck` was disabled repo-wide for a reason: it produces a large volume of findings for patterns the team has explicitly decided are acceptable (e.g., in tests, generated code, or low-risk call sites). Re-enabling it would require a broad sweep to suppress or fix hundreds of existing call sites before the linter is usable, which is disproportionate to the narrow risk being addressed here.

#### Alternative 2: Apply `//nolint:errcheck` suppressions at existing call sites

The two existing violations in `pkg/cli/logs_format_compact.go` could be addressed by simply suppressing the warning or adding a comment explaining the discard. This would fix the immediate symptom without adding enforcement for future code. However, it provides no protection against new `Flush()` call sites that silently discard errors, leaving the underlying error class undetected going forward.

### Consequences

#### Positive
- Future `Flush()` calls that discard errors will be caught at analysis time, preventing a class of silent data-loss bugs.
- The two existing violations in `pkg/cli/logs_format_compact.go` are fixed: flush errors are now logged via the package logger.
- Follows the established pattern of the linter suite: targeted, low-false-positive, type-aware analysis.

#### Negative
- Authors adding new `Flush()` call sites must explicitly handle or suppress the error, adding a small amount of boilerplate.
- If a caller intentionally ignores a flush error (e.g., because the underlying writer is known to be infallible), a `//nolint:uncheckedflushreturn` comment is required, which adds noise at those call sites.

#### Neutral
- The analyzer is registered in `allAnalyzers` and runs as part of the standard `make golint-custom` pass, so no additional CI configuration is required.
- The linter count in `doc.go` and `README.md` is bumped from 61 to 62, keeping documentation in sync with the implementation.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
