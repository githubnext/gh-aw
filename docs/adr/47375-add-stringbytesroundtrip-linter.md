# ADR-47375: Add stringbytesroundtrip Custom Go Analysis Linter

**Date**: 2026-07-22
**Status**: Draft
**Deciders**: pelikhan, linter-miner automation

---

### Context

The repository uses a custom Go static analysis framework (`pkg/linters/`) to enforce code-quality rules beyond what standard tooling catches. Go developers occasionally write `string([]byte(s))` when `s` is already a `string`, or `[]byte(string(b))` when `b` is already a `[]byte`. These round-trip conversions cause two unnecessary memory allocations and produce no semantic difference — the output type is identical to the input type. This pattern can be introduced silently during refactoring or copy-paste and is not caught by any existing linter in this repository.

### Decision

We will add a new custom Go analysis pass, `stringbytesroundtrip`, to the linter registry (`pkg/linters/registry.go`). The analyzer traverses the AST for nested `CallExpr` nodes, checks type information to confirm the outer and inner conversions form a round-trip (`string→[]byte→string` or `[]byte→string→[]byte`), and reports a diagnostic with the redundant expression identified. It integrates with the existing `nolint`, `filecheck`, and `inspect` analysis passes already used by sibling linters.

### Alternatives Considered

#### Alternative 1: Rely on an External Linter (e.g., staticcheck, revive)

Mainstream linters like `staticcheck` and `golangci-lint` could potentially cover this pattern, or a rule could be requested upstream. However, no existing external linter currently detects this exact round-trip. Relying on an external tool would delay detection, introduce a dependency on upstream acceptance timelines, and diverge from the repository's established approach of writing custom analyzers for project-specific patterns.

#### Alternative 2: Catch via Code Review Alone

The team could document this anti-pattern in style guides and rely on reviewers to flag it. This is low-cost to implement but inconsistent — the pattern is subtle enough that reviewers commonly miss it during busy review cycles. It also provides no automated enforcement at CI time and does not scale as the codebase grows or contributor count increases.

### Consequences

#### Positive
- Redundant `string([]byte(s))` and `[]byte(string(b))` patterns are caught automatically at CI time, preventing unnecessary memory allocations from reaching the main branch.
- The implementation follows the established custom-analyzer pattern in `pkg/linters/`, keeping the linter ecosystem consistent and easy to maintain.

#### Negative
- Every new analyzer added to the registry increases overall CI compilation and analysis time, albeit marginally for a single focused pass.
- Contributors unfamiliar with the round-trip anti-pattern may encounter confusing lint errors until they learn why such conversions are redundant.

#### Neutral
- The analyzer skips generated files (via `filecheck`) and respects `//nolint:stringbytesroundtrip` directives, consistent with all other custom linters in this repository.
- Test coverage is provided via `analysistest`, which is the standard testing approach for Go analysis passes in this codebase.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
