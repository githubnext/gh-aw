# ADR-54576: Split workflow_builder.go into Domain-Focused Modules

**Date**: 2026-08-21
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

`pkg/workflow/workflow_builder.go` had grown beyond 1,000 lines of code and was handling multiple distinct responsibilities: orchestrating the top-level workflow build, extracting and merging model cost/policy overlays, merging steps/pre-steps/post-steps across imported and main workflows, and extracting frontmatter helpers (enclaves, LSP, skills/plugins, concurrency, dispatch, inlined imports, excluded env). This multi-responsibility structure made it harder to identify ownership, reason about correctness, and make safe changes — any edit risked touching unrelated logic in the same file.

### Decision

We will split `pkg/workflow/workflow_builder.go` into four domain-focused files within the same `pkg/workflow` package, preserving all existing function signatures and call-graph semantics:

- `workflow_builder.go` — top-level orchestration only (`buildInitialWorkflowData`)
- `workflow_builder_model_overlays.go` — model cost/policy extraction and overlay merge logic
- `workflow_builder_steps.go` — steps/pre-steps/pre-agent-steps/post-steps merge paths
- `workflow_builder_frontmatter_extract.go` — frontmatter extraction helpers

Splitting within the same package avoids any change to import paths or the public API surface while achieving focused, reviewable modules. Targeted unit tests are added alongside the new files to validate the extracted helper groups independently.

### Alternatives Considered

#### Alternative 1: Keep the Monolithic File

Continue adding to `workflow_builder.go` as-is. This is zero-effort in the short term but compounds the maintenance problem: each new feature makes the file harder to review and more likely to cause unintended coupling. The issue that triggered this refactor (`#54540`) specifically cited the size and multi-responsibility nature of the file as a change-safety concern.

#### Alternative 2: Extract into Separate Sub-packages

Move the domain logic into sub-packages (e.g., `pkg/workflow/overlays`, `pkg/workflow/steps`). This would provide stronger encapsulation via Go's package-visibility rules, but it would require changing import paths throughout the codebase, potentially expose or hide symbols in unintended ways, and adds package boundary overhead for logic that is tightly coupled to the same workflow build context. The refactor goal is focused readability, not stronger encapsulation enforcement.

### Consequences

#### Positive
- Each file has a single, clearly named responsibility — easier to locate relevant logic and assign reviewers.
- Unit tests can be written for individual helper groups (model-cost overlay merge, inlined-imports resolution, etc.) without pulling in unrelated code.
- Future additions to one concern (e.g., new step-merge logic) land in a predictably named file, reducing merge conflicts.
- No change to the public API surface: function signatures and call graph are preserved exactly.

#### Negative
- More files to navigate for anyone wanting the complete workflow-build picture; the orchestration in `workflow_builder.go` now delegates to files that must be opened separately.
- The split decision is convention-based (by responsibility), not enforced by Go package visibility — callers inside the same package can still reach across module boundaries without the compiler flagging it.

#### Neutral
- The refactor is purely structural; runtime behavior, merge semantics, and test coverage for existing code paths are unchanged.
- The `docs/adr` naming convention (ADR number = PR number) is preserved.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
