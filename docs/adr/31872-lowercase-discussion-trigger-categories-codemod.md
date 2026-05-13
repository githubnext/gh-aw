# ADR-31872: Lowercase Discussion Trigger Categories via Default Codemod

**Date**: 2026-05-13
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

`gh-aw` normalizes discussion trigger category values (under `on.discussion.types` and `on.discussion_comment.types`) to lowercase at compile time, but the workflow source files are not automatically aligned with that normalization. As a result, source frontmatter can contain mixed-case values such as `Agentic Workflows` while the compiled runtime behavior matches `agentic workflows`, producing a silent mismatch between what the author wrote and what the workflow actually triggers on. The existing `gh aw fix --write` codemod pipeline already absorbs this class of source-vs-runtime drift for other fields, so extending it with a category-lowercasing codemod is the natural remediation path. The change must preserve YAML formatting, comments, and inline-vs-block list styles to keep diffs minimal.

### Decision

We will add a new default-on codemod `discussion-trigger-categories-lowercase` to the `GetAllCodemods()` registry. The codemod inspects `on.discussion.types` and `on.discussion_comment.types`, and when it finds any string value that is not already lowercase, rewrites the source frontmatter in place using a line-based transform that handles both inline arrays (`types: [General]`) and block sequences (`- Agentic Workflows`). The transform preserves indentation, quoting style (bare, single-quoted, double-quoted), and trailing comments, and is a no-op when all values are already lowercase. Expression values (`${{ ... }}`) are skipped to avoid corrupting templated input.

### Alternatives Considered

#### Alternative 1: Compile-Time Error or Warning Only

Reject mixed-case discussion category values at compile time and require authors to fix them manually. This was rejected because it shifts the migration burden onto every workflow author for every affected file, and because the project already prefers automated codemods over manual remediation for normalization-style migrations (see, for example, the strict-mode secret remediation codemods in ADR-26919).

#### Alternative 2: Structural YAML Parse-and-Serialize

Parse the frontmatter into a structured YAML representation, mutate the type values, and re-serialize. This is more robust against unusual indentation or multi-line scalars. It was rejected because round-trip serialization typically loses comments, blank lines, and quoting style, and every other codemod in this registry uses a line-based transform — consistency and formatting preservation outweigh the marginal robustness gain for this narrow field.

#### Alternative 3: Opt-In Flag

Gate the codemod behind a `--lowercase-discussion-categories` flag rather than including it in the default registry. This was rejected because the codemod is a strict no-op when source is already lowercase, has no destructive side effects beyond the targeted fields, and keeping it opt-in would perpetuate the source/runtime drift it was designed to eliminate.

### Consequences

#### Positive
- Workflow source frontmatter matches the compile-time normalized form, eliminating the `Agentic Workflows` vs `agentic workflows` class of confusion.
- The codemod runs automatically in the existing `gh aw fix --write` flow with no new flags or authoring steps required.
- Line-based transform preserves indentation, comments, and inline-vs-block list styles, keeping diffs minimal.
- Templated expression values (`${{ ... }}`) inside list items are skipped, so expression-driven category configuration is not corrupted.

#### Negative
- Line-based parsing of `on.discussion.types` is more brittle than a full YAML round trip; pathological indentation or unusual scalar styles outside the tested cases may not be rewritten correctly.
- The rewrite happens silently as part of the broader `gh aw fix` pipeline, so authors who intentionally used mixed case (for documentation/readability) will see their casing changed without a separate prompt.
- Adds another entry to the ordered codemod registry that must be maintained alongside existing pre-frontmatter and frontmatter-aware codemods, and that ordering is asserted by `expectedCodemodOrder()` in tests.

#### Neutral
- The codemod is registered between `add-comment-discussion-removal` and `mcp-mode-to-type-migration` in `GetAllCodemods()`; the order is asserted in `fix_codemods_test.go` and must be updated in lockstep with any registry reordering.
- The implementation uses the shared `applyFrontmatterLineTransform` helper and the existing `getIndentation` / `isTopLevelKey` utilities, so it inherits any future improvements (or regressions) to those helpers.
- `IntroducedIn` is stamped as `1.0.0`, matching the version-stamp convention used by codemods that ship as part of the initial stable codemod set rather than a later minor release.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Codemod Registration

1. The codemod **MUST** be registered in `GetAllCodemods()` with the ID `discussion-trigger-categories-lowercase`.
2. The codemod **MUST** be enabled by default and **MUST NOT** require an opt-in flag.
3. The codemod's position in the registry **MUST** be reflected in `expectedCodemodOrder()` so test assertions remain accurate.
4. The codemod **MUST** set `IntroducedIn` to a non-empty semantic version string.

### Transformation Scope

1. The codemod **MUST** lowercase string values found under `on.discussion.types` and `on.discussion_comment.types`.
2. The codemod **MUST NOT** modify values under any other top-level key, including other `on.*` triggers.
3. The codemod **MUST** handle both block-sequence list items (`- Agentic Workflows`) and inline-array list items (`types: [General]`).
4. The codemod **MUST NOT** modify list items whose value contains a GitHub Actions expression (`${{ ... }}`).
5. The codemod **MUST** preserve the original quoting style of each value (bare, single-quoted, or double-quoted).
6. The codemod **MUST** preserve trailing comments on the same line as a modified value.
7. The codemod **MUST** preserve original indentation of every line, modified or not.

### Idempotence and No-Op Behavior

1. When all targeted values are already lowercase, the codemod **MUST** return the input content unchanged and **MUST** report `applied = false`.
2. When the frontmatter has no `on` key, no `discussion` / `discussion_comment` trigger, or no `types` field, the codemod **MUST** be a no-op.
3. Running the codemod twice on the same input **MUST** produce identical output, and the second run **MUST** report `applied = false`.

### Error Handling

1. The codemod **MUST NOT** return an error for inputs that are valid frontmatter, even when no transformation applies.
2. The codemod **SHOULD** log a single message when it applies a transformation, and **SHOULD NOT** log when it is a no-op.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25781271677) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
