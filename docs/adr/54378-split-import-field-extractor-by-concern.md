# ADR-54378: Split import_field_extractor.go into Concern-Focused Files

**Date**: 2026-08-21
**Status**: Accepted
**Deciders**: pelikhan, app/copilot-swe-agent

---

### Context

`pkg/parser/import_field_extractor.go` had grown to 1,048 lines and 51 functions, all in a single file within the `parser` package. The functions implemented five distinct concerns (engine config, activation/auth fields, scalar/builder config, step/job fields, and model normalization) that were interleaved without structural separation. A DeepReport monolithic-file analysis flagged it as a high-value refactor target. Go's package model allows a single package to span multiple files with full access to unexported identifiers, making a file-level split feasible without any API changes.

### Decision

We will split `import_field_extractor.go` into five files along the existing semantic boundaries already implied by function groupings and doc-comment sections, with zero behavior change. All functions are moved verbatim; per-file import lists are trimmed to only what each file uses. The resulting files are:

- `import_field_extractor.go` — core `importAccumulator` struct, constructor, main pipeline (`extractAllImportFields`, `prepareFrontmatter`), and result building
- `import_field_extractor_engine.go` — engine config and scalar/builder fields (max-turns, mcp-servers, safe-outputs, sandbox mounts, etc.)
- `import_field_extractor_activation.go` — activation/auth fields (bots, skip-roles/bots, github-token, github-app, checkout)
- `import_field_extractor_steps.go` — step/job/env/labels/cache/features/run-install-scripts/observability extraction
- `import_field_extractor_models.go` — `models` field normalization (aliases, allow/block policies, cost overlays)

### Alternatives Considered

#### Alternative 1: Reorganize with comment blocks inside the single file

Add clearly labeled comment banners (`// ---- Engine config ----`) to group related functions in-place. This preserves contiguous `git blame` history and has zero structural change. However, it does not reduce the cognitive overhead of loading the full 1,000-line file to locate one function group, and IDE navigation still requires scrolling rather than opening a named file.

#### Alternative 2: Extract into sub-packages under pkg/parser/

Create `pkg/parser/extractor/engine`, `pkg/parser/extractor/activation`, etc. This provides the strongest isolation: each sub-package can be tested independently and dependencies are explicit. However, the `importAccumulator` struct is intentionally unexported. Elevating it to a sub-package-visible type would require either exporting it (expanding the public API) or moving the struct into the sub-package (restructuring the main file's data model). Neither change is behavior-neutral. The benefits of sub-package isolation are therefore not available at this stage without additional refactoring scope.

### Consequences

#### Positive
- Navigating to a specific concern (e.g., model alias normalization) now requires opening one clearly-named file rather than scanning a 1,000-line file.
- Future additions or removals of field extractors are scoped to a single file, reducing the chance of merge conflicts between concurrent changes.
- Each file's import list accurately documents which standard-library packages that concern depends on.

#### Negative
- The `importAccumulator` struct is defined in `import_field_extractor.go` and is an implicit shared dependency across all five files; this coupling is invisible at the call site and must be understood holistically.
- `git log -- pkg/parser/import_field_extractor.go` no longer surfaces history for functions moved into the new files; `git log --follow` or per-file blame is needed to trace pre-split history.

#### Neutral
- No callers outside the `parser` package are affected; all moved symbols remain unexported.
- Test coverage does not change: existing tests exercise the package API, not individual file boundaries.

---

*Implemented in PR #54378.*
