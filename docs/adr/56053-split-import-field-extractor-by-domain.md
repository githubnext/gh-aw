# ADR-56053: Split Import Field Extractor by Domain

**Date**: 2026-08-26
**Status**: Draft
**Deciders**: Unknown

---

### Context

`pkg/parser/import_field_extractor.go` had grown to over 1,000 lines by accumulating unrelated extraction concerns in one file: engine config, MCP settings, activation/auth fields (bots, skip-rules, GitHub App/token, checkout), step/job/env fields, model aliases and policies, and shared JSON/YAML helpers. The mixed concerns made it difficult to locate a specific extractor, reason about side effects in isolation, or align test files with the code they cover. The project has an established pattern (seen in `55482-split-safe-outputs-handler-registry-by-domain.md`, `55703-split-over-length-compiler-functions.md`) of splitting over-length files by concern rather than by layer.

### Decision

We will split `import_field_extractor.go` into five focused files within the same `parser` package, each owning one extraction domain:

- `import_field_extractor.go` — BFS orchestration, frontmatter prep/substitution, `importAccumulator` definition, `toImportsResult`/`buildImportsResult`, path normalization
- `import_field_extractor_engine.go` — engine config, MCP timeout extraction, config scalar/builder fields, sandbox mount/runtime-install merge
- `import_field_extractor_activation.go` — activation fields (`bots`, skip-*), token/app extraction, checkout, GitHub App JSON validation
- `import_field_extractor_jobfields.go` — steps/jobs/env conflict detection, features/cache/labels/excluded-env, observability, runtime install-script detection
- `import_field_extractor_models.go` — model aliases/policies/providers normalization and warnings
- `import_field_extractor_helpers.go` — shared JSON/YAML field append and merge utilities used across domains

The split preserves the existing package-private API (`importAccumulator` and all method signatures) and introduces no behavioural changes. Test files are reorganised to mirror the new source layout.

### Alternatives Considered

#### Alternative 1: Sub-package decomposition

Move each domain into its own sub-package (e.g., `pkg/parser/activation`, `pkg/parser/engine`). This is the standard Go approach for separating concerns at scale.

Rejected because `importAccumulator` and most of its collaborating types are intentionally package-private. Extracting them into sub-packages would require exporting them, widening the public API surface in a way the existing callers do not need and the package boundary does not warrant at this scale.

#### Alternative 2: In-place sectioning with region comments

Keep all code in a single file but add `// region` comments, separator banners, and a more thorough doc comment index at the top to group the five concerns visually.

Rejected because region comments are a convention, not enforced by the toolchain, and do not reduce the cognitive load of navigating a 1,000-line file. They also do not improve test co-location or per-domain diff clarity in code review.

### Consequences

#### Positive
- Each file is now focused on one extraction domain, making it easier to locate a specific method and reason about its scope.
- Test files mirror source file layout, so coverage gaps and domain responsibilities are immediately visible.
- Future additions to one domain (e.g., a new activation field) touch exactly one source file, reducing review noise.
- Aligns with the project's established file-diet pattern (see ADRs 55482, 55703).

#### Negative
- Finding a function by name now requires knowing which domain file owns it; contributors unfamiliar with the split must grep or rely on IDE navigation.
- The number of files in `pkg/parser/` increases by five source files and six test files, which adds minor build-graph breadth.

#### Neutral
- No changes to the public package API or the `importAccumulator` struct fields — callers outside the package are unaffected.
- Existing test coverage is preserved; tests are redistributed across the new files rather than rewritten.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
