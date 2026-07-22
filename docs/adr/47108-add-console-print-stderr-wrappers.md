# ADR-47108: Add `Print*` Stderr Convenience Wrappers to the `console` Package

**Date**: 2026-07-22
**Status**: Draft
**Deciders**: Unknown

---

### Context

The `pkg/console` package exposed `Format*` helpers that return formatted strings suitable for diagnostic stderr output, but provided no direct write API. Every call site was required to compose `fmt.Fprintln(os.Stderr, console.Format*(...))` — a two-step pattern that was widespread (38+ call sites in `engine_secrets.go`, 29+ in `mcp_inspect_mcp.go` alone) and inconsistently expressed: some callers used `fmt.Fprintln`, others used `fmt.Fprintf(os.Stderr, "\n%s\n", ...)` with an explicit newline wrapper, creating divergent output-routing ergonomics within the same package boundary. Callers also required `fmt` and `os` imports solely to route console output to stderr.

### Decision

We will add a thin `Print*` layer in `pkg/console/print.go` that wraps each `Format*` helper with `fmt.Fprintln(os.Stderr, ...)`, providing a single-call API for every diagnostic output variant (success, info, warning, error, command, progress, prompt, verbose, section header, list item, error chain). We will migrate the two highest-volume call sites (`pkg/cli/engine_secrets.go` and `pkg/cli/mcp_inspect_mcp.go`) to use the new wrappers immediately, and update package docs and the README to prefer `Print*` for diagnostic stderr output.

### Alternatives Considered

#### Alternative 1: Enforce the existing two-step pattern via a linter rule

The current `fmt.Fprintln(os.Stderr, console.Format*(...))` pattern could be mandated through a custom lint rule that rejects bare `fmt.Fprintln(os.Stderr, ...)` in favor of the canonical form. This avoids introducing a new API surface and keeps the format and write concerns separate. It was rejected because it adds lint infrastructure without eliminating boilerplate or import overhead, and it cannot address the divergent `Fprintf` + newline-wrapper variant already present in the codebase.

#### Alternative 2: Change `Format*` functions to write directly to stderr

The existing `Format*` functions could be converted to print side-effectfully rather than returning strings. This would achieve a single-call API without a new parallel set of names. It was rejected because it would destroy the composability of the format layer: callers that redirect output to a non-stderr writer (e.g., in-memory buffers, test writers, or `io.Writer` parameters) would lose that flexibility. Keeping `Format*` as pure string formatters and introducing separate `Print*` writers preserves both use cases.

### Consequences

#### Positive
- Call sites that only write to stderr can drop `fmt` and `os` imports, reducing import churn across CLI code.
- Stderr routing is centralized in `pkg/console` rather than distributed across every call site, making future output-redirection changes (e.g., pluggable writers) a one-place edit.
- Inconsistent newline-wrapping patterns (`Fprintln` vs. `Fprintf("\n%s\n", ...)`) are resolved; `Print*` always adds exactly one trailing newline.
- Test coverage for the new layer is provided via `pkg/console/print_test.go`.

#### Negative
- Two parallel APIs now exist for the same underlying concepts: `Format*` (string) and `Print*` (stderr side-effect). Callers must decide which to use and may mix them inconsistently during the partial migration period.
- `Print*` functions hardcode `os.Stderr` as the destination, which limits testability without OS-level pipe tricks (as demonstrated in `print_test.go`). A future `io.Writer`-parameterised API may be needed for thorough unit testing.
- The migration is intentionally incomplete: only two files are updated in this PR. The remaining call sites continue using the old pattern, so the codebase is temporarily in a mixed state.

#### Neutral
- The `Print*` functions propagate `(int, error)` return values from `fmt.Fprintln`, consistent with Go conventions, but callers universally discard both values (`_, _ = console.Print*(...)`) since stderr-write errors are not actionable at runtime.
- Package documentation (`doc.go`, `README.md`) is updated to recommend `Print*` as the preferred pattern for diagnostic output.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
