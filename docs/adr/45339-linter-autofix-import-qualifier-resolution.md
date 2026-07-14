# ADR-45339: Resolve Import Qualifier at Fix Emission Time via Shared AST Helpers

**Date**: 2026-07-14
**Status**: Draft
**Deciders**: Unknown (automated draft from PR #45339)

---

### Context

Three autofix linters (`sprintfint`, `writebytestring`, `bytescomparestring`) previously hardcoded the default package name in their `SuggestedFix` replacement text (e.g., always emitting `bytes.Equal(…)`, `strconv.Itoa(…)`, `io.WriteString(…)`). When the target package was already imported under an alias in the analysed file, the emitted fix referenced an undefined identifier and produced a non-compiling result. A closely related failure: when the default package name happened to match a local variable or parameter name (e.g., a parameter `bytes []byte`), adding the import and emitting `bytes.Equal(…)` silently resolved to the parameter instead of the package, again producing invalid code. Both failure modes were invisible to the linter's own test suite because the existing test fixtures did not cover aliased or shadowed imports.

### Decision

We will add two shared helpers to `pkg/linters/internal/astutil` and update all three affected linters to call them before constructing `SuggestedFix` content:

1. **`ImportedAs(file *ast.File, importPath string) (string, bool)`** — returns the local binding name for an import path by inspecting the file's `ast.ImportSpec` list. Returns the explicit alias when present, otherwise the last path segment as the default name. Returns `"."` / `"_"` as-is so callers can reject dot- and blank-imports.
2. **`QualifierShadowed(pkg *types.Package, pos token.Pos, name string) bool`** — uses `pkg.Scope().Innermost(pos)` and `LookupParent` to detect whether `name` is already bound to a non-package object (variable, parameter, constant) at the call site.

Each linter now (a) resolves the actual local binding name before building the replacement string, and (b) suppresses the `SuggestedFix` entirely when the qualifier is a dot-import, blank-import, or shadowed — while still reporting the diagnostic so the author is informed of the issue.

### Alternatives Considered

#### Alternative 1: Suppress all fixes for files that use aliased imports

Treat any alias on the target package as a hard skip for fix emission. This is simpler — no new helpers needed — but it silently degrades fix coverage for a common pattern (many Go codebases alias `io/ioutil` → `io`, or shorten long package names). Users would see diagnostics with no actionable fix even when one is mechanically safe.

#### Alternative 2: Delegate qualifier resolution to a post-fix goimports pass

Emit fixes with the canonical package name regardless of the current import state, and rely on a subsequent `goimports` or `gopls` pass to re-resolve qualifiers. This would handle import re-organisation automatically but cannot eliminate the shadowing problem (a local `bytes` parameter takes precedence over any import), and introduces a hard dependency on an external tool being run after the linter. It also produces intermediate non-compiling states, which breaks single-step apply workflows.

### Consequences

#### Positive
- Autofix suggestions now produce compilable output when the target package is imported under an alias, eliminating the `undefined: <pkg>` error class.
- Both shared helpers (`ImportedAs`, `QualifierShadowed`) live in the internal `astutil` package and are immediately reusable by any future linter that needs to emit qualified references, reducing per-linter boilerplate.

#### Negative
- When the qualifier is shadowed by a local variable or parameter, the linter emits a diagnostic but no `SuggestedFix`. Authors will see a warning without an accompanying one-click fix, which may be surprising if they are accustomed to all diagnostics from this linter being auto-applicable.
- Each affected linter must iterate `pass.Files` (O(files per package)) to locate the `ast.File` containing the flagged expression. For packages with many files this is a small repeated cost; it could be avoided by caching the file lookup, but that complexity is not warranted at current scale.

#### Neutral
- The diagnostic message text remains unchanged (still references the canonical package name, e.g., `bytes.Equal`), so the human-readable hint is not affected by aliasing. Only the machine-applicable `SuggestedFix` changes.
- Test fixtures (`aliased_import.go` and `shadow.go`) were added to each of the three linters' `testdata` directories, expanding the test surface area but requiring no changes to the test runner infrastructure.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
