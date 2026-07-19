# ADR-46465: Add trimleftright Linter to Detect TrimLeft/TrimRight Cutset Misuse

**Date**: 2026-07-18
**Status**: Accepted
**Deciders**: pelikhan (PR author)

---

### Context

`strings.TrimLeft(s, cutset)` and `strings.TrimRight(s, cutset)` treat the second argument as a **set of individual characters** to remove, not as a literal prefix/suffix string. When a developer passes a multi-character string literal (e.g., `strings.TrimLeft(s, "foo")`), they almost always intend to strip the exact prefix `"foo"`, but the function instead strips any leading character that is `'f'` or `'o'`. This is a well-known Go gotcha that causes silent data corruption—the program compiles cleanly and may appear to work for many inputs while quietly producing wrong results on others. The gh-aw project maintains its own linter framework under `pkg/linters/` and has an established pattern for adding static analysis passes with automated `SuggestedFix` rewrites.

### Decision

We will add a new `trimleftright` analyzer at `pkg/linters/trimleftright/` with a conservative heuristic: flag `strings.TrimLeft`/`strings.TrimRight` only when the cutset is a string literal that is (1) multi-rune, (2) fully alphanumeric, and (3) contains at least one repeated rune (for example `"foo"` or `"barr"`). This sharply reduces false positives for idiomatic character-set trimming such as whitespace classes (`" \t\n\r"`), punctuation sets, or unique-rune sets (`"aeiou"`).  

The analyzer will emit diagnostics only (no `SuggestedFix`) because replacing `TrimLeft`/`TrimRight` with `TrimPrefix`/`TrimSuffix` changes semantics and cannot be proven safe from syntax alone. Generated files are excluded and callers can suppress intentional cases with `//nolint:trimleftright`. The analyzer is registered in `cmd/linters/main.go` alongside existing analyzers.

### Alternatives Considered

#### Alternative 1: Rely on golangci-lint or an external linter

`golangci-lint` and the broader Go static analysis ecosystem could be configured to catch this pattern. The [`gocritic`](https://github.com/go-critic/go-critic) linter includes a `strings.TrimLeft/TrimRight` check. This would avoid writing and maintaining custom code.

Not chosen because: gh-aw's internal linter framework provides tight integration with the project's `nolint` infrastructure, `filecheck` (skip generated files), and `astutil` helpers. Upstream linters may not emit `SuggestedFix` entries usable by the project's automated fix workflow, and adding an external dependency for a single check adds version-pinning overhead.

#### Alternative 2: Document the gotcha in a style guide without automated enforcement

A note in the contributing guide or a code review checklist item could raise awareness of this pitfall. This requires zero implementation effort.

Not chosen because: documentation-only approaches do not catch new instances at review time and leave existing code silently broken until a human notices. Static analysis enforcement is consistent, scalable, and does not depend on reviewer familiarity with the edge case.

### Consequences

#### Positive
- Catches a silent data-corruption class at static analysis time before code reaches production.
- Emits a machine-applicable `SuggestedFix`, enabling automated rewrites via `go tool vet -fix` or editor integrations.
- Follows the established `pkg/linters/` pattern, keeping the codebase consistent.

#### Negative
- Any intentional multi-character cutset usage (e.g., trimming a set of punctuation characters) must be explicitly annotated with `//nolint:trimleftright`, adding a small maintenance burden for valid usages.
- The analyzer adds one more entry to `cmd/linters/main.go` and must be kept in sync with future refactors of the linter registry.
- The conservative heuristic may miss some genuine mistakes (for example, literal prefixes without repeated runes such as `"bar"`).

#### Neutral
- The linter applies only to string literals; dynamic cutset values (variables, expressions) are not flagged, which is a deliberate scope limitation—inferring intent from non-literal arguments is not feasible at analysis time.
- Test fixtures follow the `analysistest` golden-file convention already used by sibling linters, so no new test infrastructure is needed.

---
