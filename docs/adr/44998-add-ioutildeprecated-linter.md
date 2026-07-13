# ADR-44998: Add ioutildeprecated Linter to Flag Deprecated io/ioutil Usage

**Date**: 2026-07-12
**Status**: Draft
**Deciders**: pelikhan, linter-miner automation

---

### Context

The `io/ioutil` package was deprecated in Go 1.16 (February 2021). All its functions are thin wrappers around equivalents in the `io` and `os` packages (e.g., `ioutil.ReadAll` → `io.ReadAll`, `ioutil.ReadFile` → `os.ReadFile`). Using the deprecated package adds unnecessary indirection, signals outdated code, and may confuse readers unfamiliar with the deprecation. The repository uses a custom in-house static analysis framework (`golang.org/x/tools/go/analysis`) to enforce codebase conventions. A scan of `pkg/` and `cmd/` found no existing usages, making this a preventive measure to ensure new code does not introduce deprecated API calls.

### Decision

We will add a new custom Go analysis linter, `ioutildeprecated`, that flags any call-site reference to functions in the `io/ioutil` package and reports the modern replacement (`io.ReadAll`, `os.ReadFile`, etc.). The linter integrates with the existing in-house linter runner (`cmd/linters/main.go`) and follows the established pattern of custom analyzers in `pkg/linters/`. It skips test files and respects `nolint` directives for escape hatches.

### Alternatives Considered

#### Alternative 1: Rely on staticcheck's SA1019 rule

`staticcheck` detects usage of deprecated symbols, including all `io/ioutil` functions, via its `SA1019` (deprecated API usage) diagnostic. It is already well-known in the Go ecosystem and requires no custom code.

This was not chosen because the repository maintains its own linter suite for fine-grained control over error messages, nolint escape-hatch semantics, and integration with the internal `filecheck`/`nolint`/`astutil` utilities. Adding another external tool would fragment the linting surface and require separate CI configuration. The in-house linter can produce a more actionable message (naming the exact replacement) and follows the already-established contribution pattern for this codebase.

#### Alternative 2: Use golangci-lint with depguard or forbidigo

`golangci-lint` aggregates many linters and offers `depguard` (package import deny-listing) and `forbidigo` (symbol pattern banning) that could block `io/ioutil` usage with configuration only, requiring no new Go code.

This was not chosen for the same reason as Alternative 1: the project has chosen to own its linter implementations rather than depend on an external aggregator. Additionally, `depguard` operates at the import level and would block the import even in the test fixture files, requiring additional exclusion configuration; the custom linter already handles this by skipping test files explicitly.

### Consequences

#### Positive
- New production code calling deprecated `io/ioutil` APIs is caught at CI time before merging, preventing future technical debt.
- The linter message names the exact modern replacement function, making remediation obvious without requiring the developer to look up the deprecation notice.
- Follows the existing contribution pattern for in-house linters, keeping the codebase consistent.

#### Negative
- A new linter package must be maintained; if Go adds or removes `io/ioutil` symbols in a future version, the `replacements` map requires manual updates.
- The linter only catches selector expressions (`ioutil.X`) resolved through the type system; unusual import aliases (e.g., `import ioutil2 "io/ioutil"`) are handled correctly by the type-based check, but the dependency on `pass.TypesInfo` means incomplete type information (e.g., in ill-formed packages) silently skips the check.

#### Neutral
- The linter is purely preventive: no existing usages were found in the repository at the time of addition, so there are no immediate remediation tasks.
- The `ioutil.Discard` and `ioutil.NopCloser` symbols are variables/functions (not just functions), but the selector-expression traversal handles them uniformly with the same detection logic.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
