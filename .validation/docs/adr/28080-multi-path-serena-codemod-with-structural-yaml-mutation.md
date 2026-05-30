# ADR-28080: Multi-Path Serena Config Discovery and Structural YAML Mutation in Codemod

**Date**: 2026-04-23
**Status**: Draft
**Deciders**: pelikhan

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `serena-tools-to-shared-import` codemod automates migration of inline Serena tool configuration to the shared `shared/mcp/serena.md` import. The original implementation only detected Serena config at the top-level `tools.serena` YAML path. However, workflows that specify a custom LLM engine can place the same config under `engine.tools.serena` instead. When such workflows are processed by the old codemod, the migration is silently skipped, leaving the workflow non-compilable after the shared import becomes mandatory. A second compounding issue: workflows pinned to a specific `github/gh-aw` commit SHA via `source:` reference an older version of the import chain that predates the required `with.languages` input, so even a successfully migrated workflow would fail validation at the pinned version.

### Decision

We will extend `findSerenaLanguagesForMigration` to probe both `tools.serena` (top-level) and `engine.tools.serena` (nested under an engine block), treating whichever location is populated as the migration source. We will fix YAML block insertion to scan forward to the end of the entire `engine` block before inserting the new `imports` entry, preserving sibling engine fields such as `model` and `id`. We will replace the simple top-level block removal with an indentation-aware `removeBlockIfEmpty` that avoids deleting engine blocks that retain meaningful sibling content. Finally, we will add `maybeUpdatePinnedSourceRef`, which rewrites any `source:` value pointing to `github/gh-aw` at a 40-character commit SHA to `@main` during the same migration pass, preventing stale-import validation failures.

### Alternatives Considered

#### Alternative 1: Separate Codemods per YAML Path

Define a second, independent codemod that handles `engine.tools.serena` exclusively, leaving the original `tools.serena` codemod unchanged. This keeps each codemod small and single-purpose, but requires both to be applied in the correct order and makes it easy for a caller to apply only one — leaving workflows in a half-migrated state that is harder to debug than no migration at all.

#### Alternative 2: Require Manual Migration for Engine-Scoped Serena Config

Document that `engine.tools.serena` requires manual migration and skip automation entirely. This is low risk for the codemod itself but shifts toil onto every workflow author whose workflow uses this pattern, and provides no protection against the `source:` pin problem. Given the volume of affected workflows in the repository, manual migration was not a viable path.

### Consequences

#### Positive
- Workflows using `engine.tools.serena` are now migrated automatically, closing a silent failure mode.
- The `@main` source pin rewrite prevents post-migration validation failures against stale upstream import chains that lack the now-required `with.languages` input.
- Indentation-aware block removal preserves `engine` sibling fields (`id`, `model`) at the correct YAML depth, making the transformation structurally correct for the full range of engine block shapes.

#### Negative
- `findSerenaLanguagesForMigration` now encodes a precedence rule (top-level `tools.serena` wins over `engine.tools.serena`); if a workflow has both, only the top-level value is used and the engine-scoped config is silently discarded.
- The `maybeUpdatePinnedSourceRef` rewrite is scoped to `github/gh-aw` sources pinned by commit SHA — workflows pinned to forks, tags, or other repos are unaffected, requiring separate handling if the same pin-staleness problem arises there.
- Indentation-aware YAML mutation increases the complexity of `removeBlockIfEmpty` and `hasNestedContent`, making edge cases harder to reason about without a property-based test suite.

#### Neutral
- The change touches only `pkg/cli/codemod_serena_import.go`, keeping scope narrow and the diff reviewable in a single pass.
- Regression tests added for both `engine.tools.serena` migration with sibling preservation and the `source:` SHA rewrite path.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Serena Config Discovery

1. Implementations **MUST** detect Serena configuration at `tools.serena` (top-level) before checking `engine.tools.serena`.
2. Implementations **MUST** fall back to `engine.tools.serena` when no `tools.serena` key is present in the frontmatter.
3. Implementations **MUST NOT** attempt migration when neither location contains a non-empty `languages` list.
4. Implementations **MUST NOT** merge languages from both paths; the first matching path **SHALL** be the sole source of the language list.

### YAML Block Insertion

1. When inserting the `imports` block adjacent to an `engine:` block, implementations **MUST** scan forward to the end of the full `engine` block (i.e., until the next top-level key or end-of-document) before computing the insertion point.
2. Implementations **MUST NOT** insert the `imports` block on the line immediately following the `engine:` key, as this would interleave the new block with the engine block's nested content.

### Empty-Block Removal

1. Implementations **MUST** remove a YAML block (`tools`, `engine`) only when it contains no meaningful nested content after migration — where "meaningful" means a non-empty, non-comment child line indented deeper than the block key.
2. Implementations **MUST** preserve the enclosing block (e.g., `engine:`) when sibling fields remain after the migrated sub-key (`tools`) is removed.
3. Implementations **MUST NOT** remove a block based solely on indentation depth; the check **MUST** inspect actual content.

### Pinned Source Ref Rewrite

1. When a migration is applied and `source:` is present in frontmatter, implementations **MUST** rewrite the ref to `@main` if and only if: (a) the source repo is `github/gh-aw`, and (b) the current ref is exactly a 40-character hexadecimal commit SHA.
2. Implementations **MUST NOT** rewrite `source:` values that point to a different repository, use a named branch, or use a tag.
3. Implementations **SHOULD** perform the source ref rewrite in the same migration pass as the `tools.serena` removal to keep the workflow in a consistent state after a single codemod run.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24840177385) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
