# ADR-46060: Add mapclearloop Linter to Detect Range-Delete Loops Replaceable with clear(m)

**Date**: 2026-07-16
**Status**: Draft
**Deciders**: Unknown (automated PR — pelikhan)

---

### Context

Go 1.21 introduced the built-in `clear(m)` function, which atomically clears all entries from a map in a single call. Before Go 1.21, the idiomatic way to clear a map was a range-over-map loop with a `delete` call per entry: `for k := range m { delete(m, k) }`. Codebases that have been upgraded to Go 1.21+ frequently retain the old loop pattern due to habit or incremental migration. This project maintains a suite of custom static analysis linters (in `pkg/linters/`) that enforce idiomatic Go style across the codebase. A linter for this pattern is a natural addition: it is high-signal (the pattern is unambiguous), auto-fixable, and complements the existing `mapdeletecheck` linter which targets related map/delete idioms.

### Decision

We will add a new `mapclearloop` analysis pass (`pkg/linters/mapclearloop/`) that detects range-over-map loops whose sole body statement is `delete(m, k)` targeting the same map and key variable from the range, and flags them as candidates for replacement with `clear(m)`. The analyzer emits a `SuggestedFix` enabling automatic rewriting. It will be registered in `cmd/linters/main.go` alongside all other linters in this project.

### Alternatives Considered

#### Alternative 1: Extend the existing `mapdeletecheck` linter

The `mapdeletecheck` linter already handles a related idiom. Extending it to also catch the range-delete-entire-map pattern would avoid creating a new package and keep map-related lint rules in one place.

This was not chosen because `mapdeletecheck` addresses a distinct class of problems (likely incorrect individual deletes, e.g., deleting while iterating with possible side effects). Merging two semantically different checks into one package would blur responsibility boundaries, complicate testing, and make the codebase harder to navigate. The project's existing convention is one package per linter.

#### Alternative 2: Rely on an upstream linter (e.g., golangci-lint, staticcheck, revive)

Rather than maintaining a custom analyzer, the team could wait for the broader Go tooling ecosystem to provide this check, or configure an existing tool that already detects it.

This was not chosen because the project requires tight integration with its own linter infrastructure (nolint directives, filecheck for generated file skipping, the shared `astutil` helpers, and registration in the project's own linter runner). Upstream tools may not respect project-local nolint conventions or may have false positives in the project's specific patterns. Additionally, having the linter in-tree allows immediate fixes through `SuggestedFix` and ensures the check is versioned with the codebase.

### Consequences

#### Positive
- Code that clears maps will be more concise and express intent clearly via `clear(m)`.
- The linter has a high signal-to-noise ratio: it only fires when the range body is exactly one `delete` call on the same map with the same key, verified at the type level using `go/types` (e.g., shadowed `delete` identifiers are correctly excluded).
- The `SuggestedFix` enables automatic code rewriting, reducing developer toil when fixing violations.
- The check skips generated files via `filecheck`, avoiding noisy violations in machine-generated code.

#### Negative
- Maintainers must maintain another custom linter package indefinitely.
- Existing code in repositories using this linter suite will gain new lint failures on upgrade and must be updated to use `clear(m)`.
- The linter only targets Go 1.21+ idiom; it would produce confusing violations in codebases that cannot use Go 1.21 — though in practice this project already requires a modern Go version.

#### Neutral
- The linter follows the same structural conventions as all other linters in `pkg/linters/`: `Analyzer` var, `run` function, `testdata/` fixture with `.golden` file, `analysistest` test harness.
- No new external dependencies are introduced; `golang.org/x/tools/go/analysis` is already a project dependency.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
