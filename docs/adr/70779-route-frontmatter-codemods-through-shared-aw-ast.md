# ADR-70779: Route Frontmatter Codemods Through the Shared AW Document AST

**Date**: 2026-10-21
**Status**: Proposed (implemented in PR #70779; maintainer acceptance pending)
**Deciders**: pelikhan (PR author); maintainer acceptance pending

---

## Context

`gh aw fix` runs a registry of codemods that move authored workflow frontmatter to newer schemas: renaming keys, moving fields, removing deprecated options, merging sequences, and making some lexical rewrites. Before this PR, most codemods in `pkg/cli/codemod_*.go` had their own YAML parsing and line or indentation scanners. That made migrations fragile when frontmatter used quoted keys, flow mappings (`{a: b}`), nested fields, or comments, and several of those forms were skipped. The repository already has an authored AW document AST (`pkg/workflow/aw`) used by workflow linters and a source-preserving YAML merger in `pkg/workflow/actions`. However, codemods did not use either.

## Decision

Route frontmatter codemods through the shared editing layer, `pkg/workflow/codemod`, using the authored AW document AST rather than compiled defaults. `codemod.Apply` parses the workflow and runs a transformation on a `Frontmatter` editor with path operations (`Lookup`, `Rename`, `Set`, `Remove`, `PruneEmpty`, `Move`, `MergeSequence`, `Comment`) and an `Edit` callback for arbitrary semantic changes. Successful source replacements are reparsed to keep the document and source synchronized. `RewriteText` supports deliberate lexical rewrites, including comments, within the frontmatter boundaries.

The editor combines targeted source edits with the existing Actions source-preserving merger exposed through `actions.EditSource`. Token and source-span edits retain authored syntax where supported; semantic edits and unsupported source-edit shapes use the shared merger instead of unconditionally reserializing the frontmatter. This is a shared semantic model with syntax-aware editing helpers, not a single parser or a guarantee that every changed node retains its exact spelling.

The transformation contract is:

- A callback error returns the original content, even after earlier edits in that callback. Atomicity applies to one `Apply` invocation, not the entire registry or external side effects.
- No-op transformations return byte-identical input, and frontmatter edits leave the Markdown body untouched.
- `Rename` and `Move` do not overwrite an existing destination, including an explicit null. This does not prevent intentional replacement through `Set` or `Edit`.
- Semantic edits isolate aliased values so changing one authored path does not inadvertently change another. Sequence merging appends unique source entries and removes the source field; source-preserving paths retain authored entry comments where supported.

The shared merger also handles multiline literals, flow scalar quoting, sequence comments, and key renames needed by these migrations.

## Alternatives Considered

These are design tradeoffs, not claims that alternative implementations were built or benchmarked.

### Keep separate scanners in each codemod and fix edge cases one at a time

Keeping local string and indentation logic would reduce the initial refactor and leave each codemod self-contained. It would also repeat parsing and preservation fixes across many codemods without a common contract. Centralizing these responsibilities makes a correctness fix reusable.

### Parse frontmatter to a generic map, edit it, and reserialize it

Whole-block serialization is a simpler semantic approach, but discards comments and can change key order, quoting, and formatting far beyond the migrated field. A semantic no-op check could preserve unchanged input, but changed documents would still incur broad formatting churn. The mixed source-edit and merger approach limits that churn.

### Build a separate codemod-specific YAML CST editor

A dedicated concrete-syntax editor could offer finer control over formatting, but would introduce another editing model and its preservation rules alongside the existing AW AST and Actions merger. The chosen approach reuses their semantic and merging behavior while retaining targeted syntax helpers where needed; it does not eliminate syntax-level complexity.

## Consequences

### Positive
- Codemods share path lookup, rename, move, removal, and sequence merging instead of maintaining separate scanners for each migration.
- Quoted keys, flow mappings, and nested forms that were skipped can now be migrated structurally.
- Atomic `Apply` semantics prevent partial output from a failed callback, while byte-identical no-ops and an unchanged Markdown body reduce unrelated diffs.
- Linters and codemods share the authored AW document model rather than separately interpreting compiled defaults.

### Negative
- Changed or newly inserted nodes may use canonical YAML formatting rather than the author's original style, so some migrated files will show formatting changes.
- Codemods depend on `pkg/workflow/aw` and the shared `pkg/workflow/actions` merger. Regressions in either can affect multiple migrations.
- Source offsets, YAML syntax nodes, and merger fallback paths remain subtle components requiring preservation and alias regression coverage. Reparsing after replacements adds work; this decision does not establish a performance improvement.

### Neutral
- `RewriteText` intentionally operates on frontmatter text rather than mapping paths. Its result is reparsed, but the individual migration remains responsible for selecting the intended text and semantics.
- Exact formatting and comment placement on changed or reconstructed nodes are not universal guarantees. Unchanged-source preservation and alias isolation must be maintained across both direct-edit and merger paths.
- The codemod registry order and migration metadata are unchanged. Codemod authoring guidance was updated to point to the new helpers.
