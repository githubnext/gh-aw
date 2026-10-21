# ADR-70779: Route Frontmatter Codemods Through the Shared AW Document AST

**Date**: 2026-10-21
**Status**: Draft
**Deciders**: pelikhan (PR author); maintainer acceptance pending

---

### Context

`gh aw fix` runs a registry of codemods that move authored workflow frontmatter to newer schemas: renaming keys, moving fields, removing deprecated options, merging sequences, and making some lexical rewrites. Before this PR, most codemods in `pkg/cli/codemod_*.go` had their own YAML parsing and line or indentation scanners. That made migrations fragile when frontmatter used quoted keys, flow mappings (`{a: b}`), nested fields, or comments, and several of those forms were skipped. The repository already has an authored AW document AST (`pkg/workflow/aw`) used by workflow linters and a source-preserving YAML merger in `pkg/workflow/actions`. However, codemods did not use either.

### Decision

We will make all frontmatter codemods use a shared editing layer, `pkg/workflow/codemod`. The layer is built on the authored AW document AST and the Actions source-preserving YAML merger. `codemod.Apply` parses the workflow once and runs one transformation on a `Frontmatter` editor with path operations (`Lookup`, `Rename`, `Set`, `Remove`, `PruneEmpty`, `Move`, `MergeSequence`, `Comment`). It also provides `RewriteText`, which is limited to frontmatter. `Apply` has these contracts:

- If a transformation fails, it returns the original content.
- If nothing changes, the output is byte-identical to the input.
- It never changes the Markdown body.
- An existing destination field takes precedence over a renamed or moved field.
- Semantic edits are isolated from YAML aliases.

To support these edits, `actions.EditSource` exposes the shared merger. The merger now handles multiline literals, flow scalar quoting, sequence comments, and key renames.

### Alternatives Considered

#### Alternative 1: Keep separate scanners in each codemod and fix edge cases one at a time

Each codemod would keep its own string and indentation logic, with targeted fixes for quoted keys and flow mappings. This would avoid a large refactor and keep each codemod self-contained. We rejected it because the same parsing bugs would keep appearing across more than 40 codemods. Each fix would need to be repeated, and the edge cases would stay inconsistent.

#### Alternative 2: Parse frontmatter to a generic map, edit it, and reserialize it

Codemods would unmarshal frontmatter to a `map[string]any`, change it, and marshal the whole block again. This is the simplest semantic approach. We rejected it because it would lose comments, key order, quoting style, and other formatting the author chose. It would also turn small migrations into large diffs and break byte-identical no-ops.

#### Alternative 3: Build a codemod-specific YAML CST editor separate from the linter AST

This would be a new concrete-syntax editor used only for migrations. It could preserve formatting precisely, but it would be a second YAML model. That model would need to stay in sync with the AW AST that linters and the compiler already use. Reusing the existing AST and merger keeps one representation of authored frontmatter.

### Consequences

#### Positive
- Codemods share one tested implementation of path lookup, rename, move, removal, and sequence merging. This removes several thousand lines of duplicated scanner code from `pkg/cli`.
- Quoted keys, flow mappings, and nested forms that were skipped can now be migrated structurally.
- Atomic `Apply` semantics make it safer to run codemods together: a failed transformation does not partly rewrite a file, and no-ops return identical bytes.
- Linters and codemods use the same AST, so migrations find and edit fields the same way linters report them.

#### Negative
- Changed or newly inserted nodes may use canonical YAML formatting rather than the author's original style, so some migrated files will show formatting changes.
- Codemods now depend on `pkg/workflow/aw` and on the internals of the `pkg/workflow/actions` merger. Regressions in the shared merger can affect every codemod.
- The editing layer (`frontmatter_source.go`, `source_scalar.go`) is a large, subtle offset-based component. It needs fuzz and preservation tests to stay correct.

#### Neutral
- `RewriteText` remains available for migrations that must change YAML text intentionally, limited to frontmatter. These migrations do not get structural guarantees.
- The codemod registry order and migration metadata are unchanged. Codemod authoring guidance was updated to point to the new helpers.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
