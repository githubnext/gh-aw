# ADR-52219: Decompose `parseFrontmatterSection` into Single-Purpose Helper Methods

**Date**: 2026-08-12
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

Daily `make golint-custom` runs surfaced 292 `largefunc` findings in `pkg/workflow`. The most prominent single-function offender in this slice was `parseFrontmatterSection` in `compiler_orchestrator_frontmatter.go`, clocking in at 196 lines. The function mixed file I/O, YAML parsing, frontmatter classification (shared vs. redirect-only vs. main workflow), multi-stage validation, and result construction into a single control-flow body. This violated the project's function-length lint gate and made each concern harder to read and test in isolation.

### Decision

We will decompose `parseFrontmatterSection` into four single-purpose helper methods on `*Compiler` — `readAndParseFrontmatter`, `parseSharedOrRedirectWorkflow`, `validateMainWorkflowFrontmatter` (which in turn calls `validateMainWorkflowSchemaAndEventFilters`, `validateMainWorkflowMarkdownConstraints`, and `emitMainWorkflowWarnings`) — and introduce a `createFrontmatterParseResult` factory with functional options (`withSharedWorkflow`, `withRedirectOnly`). The extraction preserves all existing behavior and validation order; no logic is changed, only reorganized.

### Alternatives Considered

#### Alternative 1: Suppress the lint warning with `//nolint:largefunc`

Add a directive to silence the linter and treat `parseFrontmatterSection` as an accepted exception. This keeps the diff minimal and avoids any refactor risk. It was rejected because the function's length genuinely impairs readability — each of the five logical stages (I/O, classification, main-workflow validation, markdown constraints, warning emission) is conceptually independent, and the lint gate exists precisely to enforce that separation.

#### Alternative 2: Extract a dedicated `FrontmatterParser` type

Move all parsing logic into a new struct (`FrontmatterParser`) in its own file, decoupling it from `*Compiler`. This would provide a stronger separation of concerns and make unit-testing individual stages easier without a full `Compiler` instance. It was rejected because it requires broader changes (updated call sites, new constructor, exported or internal type decision) that exceed the "minimal first slice" mandate from issue #52207 and risk unintended behavior changes.

### Consequences

#### Positive
- Each extracted helper has a single, named responsibility, reducing cognitive load when reading or debugging a specific stage of frontmatter parsing.
- The `largefunc` lint finding for `parseFrontmatterSection` is eliminated without any semantic changes, keeping the diff easy to review.
- The functional-options factory (`createFrontmatterParseResult` + `withSharedWorkflow` / `withRedirectOnly`) eliminates repeated struct-literal duplication across the three result-construction sites.
- A new focused regression test (`TestParseFrontmatterSection_RedirectOnlyWorkflow`) covers the redirect-only detection path that was previously untested.

#### Negative
- Tracing the full parse pipeline now requires following multiple function calls instead of reading a single top-level body; engineers unfamiliar with the codebase may need to jump between more functions to build a mental model.
- The functional-options pattern (`frontmatterParseResultOption`) introduces an abstraction layer that can be surprising to contributors who have not seen it before in this codebase.

#### Neutral
- `*Compiler` gains seven new methods, expanding its method surface area, though all are package-private.
- The change is intentionally scoped to `pkg/workflow/compiler_orchestrator_frontmatter.go` and its test file; no other packages are affected.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
