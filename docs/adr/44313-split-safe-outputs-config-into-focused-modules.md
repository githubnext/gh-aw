# ADR-44313: Split Safe-Outputs Config Parsing into Focused Modules

**Date**: 2026-07-09
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

`pkg/workflow/safe_outputs_config.go` had grown to 1,254 lines and contained a single `extractSafeOutputsConfig` function of ~760 lines that mixed configuration type definitions, frontmatter extraction orchestration, per-handler parsing helpers, bounded-integer parsing, and handler-manager config assembly. This made it difficult to navigate, understand the boundaries between concerns, and safely extend individual pieces in isolation. Bounded-integer parsing logic for `max-patch-size`, `max-patch-files`, and `timeout-minutes` was duplicated across three separate blocks with subtle inconsistencies between them.

### Decision

We will split `safe_outputs_config.go` into six focused files within the same `workflow` package, each owning a single concern:
- `safe_outputs_config_types.go` — shared config types and logger
- `safe_outputs_config_extraction.go` — top-level frontmatter extraction orchestration
- `safe_outputs_config_global.go` — global safe-outputs field parsing
- `safe_outputs_config_base.go` — shared per-handler base parsing helpers (`parseBaseSafeOutputConfig`, `parseSamplesValue`)
- `safe_outputs_config_runtime.go` — handler-manager config assembly and serialization
- Original `safe_outputs_config.go` — deleted

We will also extract a shared `parseBoundedIntField` / `parseBoundedIntFieldOrDefault` helper and reuse it for all three bounded-integer fields, eliminating the duplicated type-switch blocks.

### Alternatives Considered

#### Alternative 1: Keep the monolithic file with improved internal documentation

Add section headers and inline comments to make the single file easier to navigate without changing the file structure.

This was not chosen because it addresses only discoverability, not the underlying readability and maintainability problems. A reader still needs to hold the entire 1,254-line file in context, and adding shared helpers for bounded-integer parsing still requires the same refactor within the file. A doc-only fix would also not enforce module boundaries or make future additions land in the right place.

#### Alternative 2: Move each concern into a separate sub-package under `pkg/workflow/safeoutputs/`

Create a dedicated sub-package (`pkg/workflow/safeoutputs/`) with exported types and functions, giving each file full package-level separation.

This was not chosen because it would require exporting currently unexported symbols (types, helpers, the logger), changing call sites across the `workflow` package, and adding a circular-import risk. The benefit of true package isolation did not outweigh the cost of the broader refactor and the API surface change. Staying in the same package achieves the readability goal with minimal blast radius.

### Consequences

#### Positive
- Each file is focused on a single responsibility, making the codebase easier to navigate and understand
- Shared `parseBoundedIntField` / `parseBoundedIntFieldOrDefault` eliminates three copies of the type-switch parsing logic, reducing the risk of inconsistent handling for future bounded-integer fields
- New test coverage for the shared bounded-int helper documents and locks in truncation, clamping, and invalid-value behavior

#### Negative
- More files to open when tracing a full code path through config extraction (six files instead of one)
- The split is behavior-preserving by intent, but the refactor introduces risk that subtle behavioral differences could creep in if the split was not perfectly faithful to the original; test coverage and reviewer attention are required to catch these

#### Neutral
- All changes are internal to the `workflow` package; no public API surface or exported symbols change
- The `extractSafeOutputsConfig` function remains the single entry point for callers, so external call sites are unaffected

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
