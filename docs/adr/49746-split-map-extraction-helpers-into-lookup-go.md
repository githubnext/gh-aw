# ADR-49746: Split Map-Extraction Helpers into lookup.go

**Date**: 2026-08-02
**Status**: Accepted
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

`pkg/typeutil/convert.go` grew to contain two unrelated concerns: safe single-value numeric conversion and keyed `map[string]any` extraction. The four map-extraction functions (`ParseBool`, `LookupMap`, `LookupString`, `LookupStringPath`) lived alongside numeric helpers but were not described in the file-level package doc, creating visible documentation drift. Callers of the map-extraction API had no clear signal that these utilities existed or where to look for them.

### Decision

We will move the four map-extraction functions (`ParseBool`, `LookupMap`, `LookupString`, `LookupStringPath`) from `convert.go` into a new file `lookup.go` within the same `typeutil` package. The new file carries a focused file-level doc block describing the map-extraction cluster. `convert.go` gains a cross-reference to `lookup.go` and narrows its package doc to numeric conversions only.

This is a pure reorganization — no signatures or behavior change. All callers and tests are unaffected.

### Alternatives Considered

#### Alternative 1: Keep all functions in convert.go (status quo)

Leave `ParseBool`, `LookupMap`, `LookupString`, and `LookupStringPath` in `convert.go` alongside numeric helpers. Requires no file changes. Rejected because the mixed concerns make the file harder to navigate and the map-extraction group remains undiscoverable via the file-level doc.

#### Alternative 2: Extract map-extraction helpers into a sub-package (e.g., pkg/typeutil/lookup/)

Move the four functions into a child package, giving them their own import path. This would enforce the separation at the compiler level. Rejected because it would require import-path changes in all callers, adds package-boundary overhead (exported identifiers already have the `typeutil.` qualifier), and is disproportionate to a four-function cluster that is logically still part of the same utility layer.

### Consequences

#### Positive
- Each file in `pkg/typeutil/` has a single focused responsibility, reducing cognitive load when navigating the package.
- Map-extraction functions are discoverable via `lookup.go`'s dedicated file-level doc block.
- The `README.md` gains a dedicated **Map Extraction** section and an updated function-choice table, improving package-level documentation.

#### Negative
- The `typeutil` package now spans two files; contributors must know that map helpers live in `lookup.go`, not `convert.go`. The cross-reference comment in `convert.go` and the README mitigate but do not eliminate this discovery cost.
- Pure reorganization carries a small risk of merge conflicts if concurrent PRs add functions to `convert.go` in the moved region.

#### Neutral
- No change to the public API surface, test coverage, or runtime behavior.
- The `docs/adr/` naming convention uses the PR number as the ADR number, consistent with other ADRs in this repository.

---
