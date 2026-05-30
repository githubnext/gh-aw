# ADR-33276: Adopt Lipgloss Tree Rendering for CLI Hierarchies

**Date**: 2026-05-19
**Status**: Draft
**Deciders**: Unknown

---

## Part 1 — Narrative (Human-Friendly)

### Context

CLI output for `mcp inspect` and `status` previously mixed ad-hoc hierarchy formatting (manual indentation, custom prefixes) with table-only views. Nested relationships — a workflow's engine and MCP servers, or a workflow's dependency graph — were hard to read because they were either flattened into tables or rendered with bespoke string-building logic in each command. The codebase already standardized terminal styling under `pkg/styles` (including `TreeEnumerator` and `TreeNode`) and has adopted `charm.land/lipgloss/v2` as the terminal rendering library, so a shared tree primitive was available but not yet used for hierarchical command output.

### Decision

We will adopt `charm.land/lipgloss/v2/tree` as the canonical primitive for rendering hierarchical CLI output in `pkg/cli`, starting with `mcp inspect` (workflow → engine + MCP servers) and the verbose mode of `status` (workflow → dependencies). All such trees will use `tree.RoundedEnumerator` together with the existing `styles.TreeEnumerator` and `styles.TreeNode` styles, so hierarchical output is visually consistent across commands.

### Alternatives Considered

#### Alternative 1: Keep manual hierarchy formatting per command

Each command would continue building its own indented strings (or rely on table-only views) to convey hierarchy. Rejected because it duplicates formatting logic across commands, drifts visually as each command evolves independently, and makes nested relationships (a tree with three or more levels) hard to express without reinventing tree primitives.

#### Alternative 2: Write a small in-repo tree printer

We could add a focused helper under `pkg/console` or `pkg/styles` that prints trees with the project's style tokens, avoiding a new dependency surface. Rejected because `lipgloss/v2/tree` is already a transitive dependency (the project uses `lipgloss/v2` extensively for terminal styling), and reimplementing tree layout, enumerator handling, and styling would duplicate well-tested upstream code with no clear benefit.

### Consequences

#### Positive
- Hierarchical CLI output is rendered consistently across commands using one library and one set of styles.
- Adding a new hierarchical view in another command becomes a small, declarative composition of `tree.Root(...).Child(...)` calls instead of bespoke string formatting.
- The tree rendering is testable as a pure function (see the new `*_tree_test.go` files), which is harder to do for ad-hoc string concatenation.

#### Negative
- Adds a concrete coupling between `pkg/cli` command code and the `charm.land/lipgloss/v2/tree` API; replacing the renderer in the future would require touching every site that builds a tree.
- Tree output is written to `stderr` alongside existing info messages, which can be noisier in `mcp inspect` and in verbose `status` runs that pipe through tooling expecting a clean stderr.

#### Neutral
- The shape of `WorkflowStatus` grows a new `Dependencies []string` field (JSON-tagged `omitempty`, console-tagged `-`), so JSON consumers may now see a `dependencies` key for workflows that import or include other files.
- Dependency extraction logic now lives in `pkg/cli/status_command.go` (frontmatter `imports` in string/list/object forms plus inline `@include`/`@import` directives, with `#section` stripping, dedup, and sort); future changes to import/include syntax must update this extractor.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Tree Rendering Library

1. Hierarchical CLI output in `pkg/cli` **MUST** be rendered using `charm.land/lipgloss/v2/tree`.
2. Implementations **MUST NOT** introduce a second tree-rendering library or hand-rolled ASCII tree printer in `pkg/cli` while this ADR is in effect.
3. Tree composition **SHOULD** use `tree.Root(...).Child(...)` chains rather than mutating tree nodes after construction.

### Styling Consistency

1. Tree renderers in `pkg/cli` **MUST** apply `tree.RoundedEnumerator` as the enumerator.
2. Tree renderers **MUST** apply `styles.TreeEnumerator` as the `EnumeratorStyle` and `styles.TreeNode` as the `ItemStyle`.
3. Renderers **SHOULD NOT** override these styles inline with per-call style values; new style variants **SHOULD** be added to `pkg/styles` first.

### Output Channel and Triggering

1. Tree output for `mcp inspect` and `status` **MUST** be written to `stderr`, alongside existing informational messages, and **MUST NOT** be written to `stdout` when `stdout` is reserved for machine-readable output (e.g., JSON).
2. The dependency tree in `status` **MUST** only be rendered when verbose text output is requested; it **MUST NOT** be rendered in JSON mode or in non-verbose runs.
3. Renderers **MUST** return an empty string when there is nothing to display (e.g., zero MCP servers, zero dependencies) so the caller can skip the surrounding section header.

### Dependency Extraction (Status Command)

1. The status command **MUST** extract workflow dependencies from both frontmatter `imports` (accepting string, list, and object-with-`uses` forms) and inline `@include` / `@import` directives in the workflow body.
2. Extracted dependencies **MUST** be normalized by stripping any `#section` suffix, deduplicated, and sorted before rendering.
3. The `Dependencies` field on `WorkflowStatus` **MUST** be JSON-tagged `omitempty` so workflows without dependencies do not emit an empty list in JSON output.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26092090247) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
