# ADR-50841: Add Fix Codemods for `toolset` Typo and `allowed-repos: current` Legacy Alias

**Date**: 2026-08-06
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The daily cross-repo compatibility audit revealed two distinct, unambiguous migration gaps in `gh aw fix --write`: the `tools.github.toolset:` (singular) typo — for which the compiler already emits "Did you mean 'toolsets'?" — and the `tools.github.allowed-repos: current` legacy alias, which strict-mode rejects because only `all`, `public`, or `${{ github.repository }}` are accepted values. Both issues caused compilation failures in external repos (`githubnext/gh-aw-trial-oxpecker-test`, `pelikhan/github-agentic-workflows`) that `fix --write` could not automatically resolve. The existing codemod framework (`GetAllCodemods()`, `applyFrontmatterLineTransform`) provides a well-established pattern for exactly these line-level YAML rewrites, and both transformations are deterministic with no ambiguity about the desired output.

### Decision

We will implement two new codemods — `toolset-singular-to-toolsets` and `allowed-repos-current-to-github-repository` — and register them in `GetAllCodemods()` immediately after the existing `github-repos-to-allowed-repos` codemod. Both codemods use the existing `applyFrontmatterLineTransform` hook with indentation-aware, context-scoped line parsers that skip comments, preserve trailing comments, and are guarded by a frontmatter pre-check to remain no-ops on already-migrated files. The `allowed-repos` codemod adds `findTrailingCommentIndex` to correctly split YAML-comment boundaries from values (a `#` only starts a comment when preceded by whitespace).

### Alternatives Considered

#### Alternative 1: Accept Legacy Forms in the Compiler

Relax strict-mode validation to silently accept `toolset:` (singular) and `allowed-repos: current` as valid input, interpreting them as their canonical equivalents at parse time. This removes the need for migration codemods entirely.

Not chosen because it undermines the strict-mode design goal: strict-mode exists specifically to reject ambiguous or deprecated configurations. Silently accepting `current` would mean workflows that rely on the deprecated alias never get upgraded to the more explicit `${{ github.repository }}` form, and the compiler's "Did you mean 'toolsets'?" guidance loses its purpose.

#### Alternative 2: Require Manual Fixes, Improve Error Messages Only

Keep `fix --write` unchanged; instead, improve the compiler error messages to be copy-paste-ready (e.g., display the exact replacement YAML inline). Users manually apply the one-line change.

Not chosen because the codemod framework already exists for exactly this purpose, and "Did you mean 'toolsets'?" is already in place. Adding automation is consistent with the project's existing pattern and directly reduces friction for the 20+ repos affected across future audit runs.

### Consequences

#### Positive
- `gh aw fix --write` now closes two additional identified failure clusters from the daily cross-repo audit, reducing manual intervention needed by external repo maintainers.
- Both codemods are idempotent and scoped strictly to `tools.github` block lines, preventing false-positive rewrites in other YAML blocks.
- `findTrailingCommentIndex` correctly handles `#` characters embedded in quoted YAML values (e.g., `"#current"` alias), improving robustness of the comment-preservation logic.

#### Negative
- Two more entries in `GetAllCodemods()` increase the ordered list that tests must enumerate; any future reordering requires updating `fix_codemods_test.go` in two places.
- The line-based YAML parsing approach (rather than a full AST parser) is a simplification that can fail on unusual but valid YAML: multi-document files, block scalars, or flow-style mappings that span multiple lines. These edge cases are untested.

#### Neutral
- Both codemods are registered with `IntroducedIn: "0.85.5"`, matching the patch release expected to ship the new migrations.
- The `allowed-repos` codemod normalizes output to double-quoted `"${{ github.repository }}"` regardless of the input quoting style (`current`, `"current"`, `'current'`), which is a minor stylistic imposition but aligns with the canonical form used throughout the codebase.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
