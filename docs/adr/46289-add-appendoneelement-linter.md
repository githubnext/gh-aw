# ADR-46289: Add appendoneelement Linter to the Internal Linter Suite

**Date**: 2026-07-18
**Status**: Draft
**Deciders**: Unknown (automated PR by linter-miner bot; human review required)

---

### Context

The repository maintains a suite of custom `go/analysis` linters in `pkg/linters/` that catch Go anti-patterns not covered by default `golangci-lint` rules. The pattern `append(s, []T{x}...)` — spreading a single-element composite slice literal — allocates a temporary slice on every call, whereas the idiomatic form `append(s, x)` does not. This anti-pattern was identified by examining the existing linter corpus (particularly `appendbytestring`) and recognising that the complementary single-element-spread case was not yet covered. The rule has zero false positives by construction: it only fires when the second argument is a composite literal with exactly one element.

### Decision

We will add a new `go/analysis` linter, `appendoneelement`, to `pkg/linters/` and register it in `cmd/linters/main.go`. The analyzer traverses all `ast.CallExpr` nodes, matches calls to the built-in `append` with exactly two arguments and an ellipsis, and reports a diagnostic (with a suggested fix) when the second argument is a single-element slice composite literal. Generated files and `//nolint:appendoneelement`-suppressed sites are skipped via existing internal helpers.

### Alternatives Considered

#### Alternative 1: Manual Code Review Only

Rely on human reviewers to flag `append(s, []T{x}...)` during PR review rather than automating detection.

Considered because it requires zero implementation effort. Rejected because manual review is inconsistent, does not scale, and misses occurrences in files that are not under active review. Automated static analysis guarantees coverage on every diff.

#### Alternative 2: Contribute the Rule to an External Linter

Upstream the rule to `golangci-lint` or another external linter rather than maintaining it internally.

Considered because it would benefit the broader Go community. Rejected because the contribution cycle for external projects is slow and uncertain, and the project already has an established pattern for housing such rules internally (e.g., `appendbytestring`, `bytesbufferstring`). Internal ownership also allows faster iteration.

### Consequences

#### Positive
- Eliminates avoidable temporary slice allocations from the flagged pattern in the codebase.
- Detection is automated, consistent, and applied to every future change — reviewers need not remember to check for this pattern.
- Follows the established internal linter convention, requiring minimal ramp-up for contributors.
- The suggested fix is emitted automatically, so authors can apply the correction with a single editor action.

#### Negative
- Adds another internal linter to maintain; any future changes to the `go/analysis` API or internal helper packages (`astutil`, `nolint`, `filecheck`) will require updating this analyzer.
- Authors who have a legitimate reason to use the spread form must suppress the diagnostic with `//nolint:appendoneelement`, adding a small annotation burden.

#### Neutral
- The linter is registered alongside all other custom analyzers in `cmd/linters/main.go`, keeping the registration surface in one place.
- The rule scope is narrow (exactly two args, ellipsis, single-element composite literal), so adding it has no effect on unrelated append patterns.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
