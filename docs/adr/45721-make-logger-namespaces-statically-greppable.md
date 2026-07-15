# ADR-45721: Make Workflow Logger Namespaces Statically Greppable

**Date**: 2026-07-15
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The `pkg/workflow` package contains ~48 validation files, each declaring a package-level logger variable. These loggers were created via a shared helper `newValidationLogger(domain string)` in `validation_helpers.go`, which constructed the namespace string at runtime via concatenation: `"workflow:" + domain + "_validation"`. This made it impossible to statically discover the complete set of logger namespaces using grep (e.g., `grep -rn 'logger.New'`) because the actual string values only existed at runtime. Additionally, `close_entity_helpers.go` embedded three `logger.New(...)` calls directly inside a composite slice literal, which also evaded static grep discovery. One call in `push_to_pull_request_branch_validation.go` carried a latent bug where `newValidationLogger("push_to_pull_request_branch_validation")` silently produced the doubly-suffixed namespace `"workflow:push_to_pull_request_branch_validation_validation"`.

### Decision

We will remove the `newValidationLogger()` helper and replace every call site with a direct `logger.New("workflow:<domain>_validation")` string literal. In `close_entity_helpers.go`, the three inline `logger.New(...)` calls inside the registry slice literal will be extracted into named package-level vars. The doubled-suffix bug will be corrected to `"workflow:push_to_pull_request_branch_validation"` as part of the same pass. No behaviour changes beyond that one bug fix are introduced.

### Alternatives Considered

#### Alternative 1: Keep `newValidationLogger()` and add a static analysis lint rule

The helper is retained but a custom lint rule (or `go generate` pass) enumerates its call sites and records the resolved namespace strings in a generated file. This preserves the DRY convention enforced by the helper while still enabling static discovery.

Rejected because it adds tooling complexity (a custom linter or generator that must be maintained and run in CI) without eliminating the runtime indirection. The direct string literal approach is simpler and equally correct, and Go's import cycle prevents the namespace string from drifting silently.

#### Alternative 2: Use a code generator to produce a central namespace registry

A generator scans all `*_validation.go` files, extracts the domain names, and emits a single `logger_namespaces_gen.go` file listing all `logger.New(...)` calls. Callers import the generated constants.

Rejected as over-engineering. The benefit (one authoritative file) does not outweigh the cost (a generator to maintain, a generated file to keep in sync, and a new build step). Direct `logger.New()` literals with a well-known naming convention are greppable and self-documenting without any tooling.

### Consequences

#### Positive
- All logger namespace strings are now fully static: `grep -rn 'logger.New'` in `pkg/workflow` yields the complete, authoritative list.
- The latent doubled-suffix bug (`push_to_pull_request_branch_validation_validation`) is fixed.
- The `newValidationLogger` helper is removed, shrinking the public API surface of `validation_helpers.go`.

#### Negative
- Each of the ~48 `*_validation.go` files now carries a direct `import "github.com/github/gh-aw/pkg/logger"`, increasing per-file coupling to the logger package (previously mediated by the shared helper in one file).
- Future validation files must manually follow the `"workflow:<domain>_validation"` naming convention; the helper previously enforced this implicitly. A new file author could deviate without a compile error.

#### Neutral
- The change is a mechanical find-and-replace across 50 files; the diff is large by line count but trivially reviewable because every hunk is structurally identical.
- `close_entity_helpers.go` acquires three extra package-level var declarations moved out of the slice literal initialiser.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
