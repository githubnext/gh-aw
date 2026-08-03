# ADR-49859: Normalize source Field Position to Prevent Spurious Merge Conflicts

**Date**: 2026-08-02
**Status**: Draft
**Deciders**: pelikhan (via copilot-swe-agent)

---

### Context

`gh aw update` performs a 3-way merge (`git merge-file`) between `base`, `current` (local workflow file), and `new` (upstream). The `source:` frontmatter field is managed by the update tool and is appended at the end of frontmatter by `UpdateFieldInFrontmatter`. However, some workflow files were committed with `source:` in a non-canonical position (e.g., before `features:` or `evals:` keys). This positional mismatch had two consequences:

1. `hasLocalModifications` returned false positives, triggering merge mode when the only actual difference was `source:` field position.
2. Within `MergeWorkflowContent`, `base` and `new` had `source:` at the end (appended by `UpdateFieldInFrontmatter`), but `current` had it in a different position. This caused `git merge-file` to emit conflict markers even when the underlying content was identical, because a 1-byte trailing-newline divergence (introduced by `parser.TrimSpace` stripping the newline that `NormalizeWhitespace` adds) amplified the positional mismatch.

### Decision

We will fix the issue in two places in `pkg/cli`:

1. **`hasLocalModifications`** — strip `source:` from both sides before comparing, so position-only differences no longer count as local modifications.
2. **`MergeWorkflowContent`** — after normalising `current`, call the new `MoveTopLevelFieldToEnd(current, "source", currentSourceSpec)` to canonicalise `source:` position to match `base` and `new` (both of which have `source:` appended at end by `UpdateFieldInFrontmatter`), then apply a final `NormalizeWhitespace` to close the trailing-newline divergence. `MoveTopLevelFieldToEnd` performs the remove-and-reappend in a single reconstruction pass to avoid cascading round-trip formatting drift.

### Alternatives Considered

#### Alternative 1: Normalize source position at write time

Rewrite `gh aw add` (and any other command that writes workflow files) to always position `source:` last. Migrate existing files via a codemod on the next `gh aw update` run.

This addresses the root cause at the source, but requires a migration pass across all installed workflow files and complicates the write path. It also does not fix files already committed in the non-canonical position until they are updated.

#### Alternative 2: Strip source from all three versions before merging

Remove `source:` from `base`, `current`, and `new` before invoking `git merge-file`, then re-inject the correct `source:` value into the merged result afterward.

This is simpler (no repositioning logic) but requires post-merge injection and makes it harder to reason about what `git merge-file` receives. It also changes the contract of `MergeWorkflowContent` more substantially, affecting callers that depend on `source:` surviving the merge unchanged.

### Consequences

#### Positive
- `gh aw update` no longer produces spurious conflict markers when `source:` is not at the end of frontmatter in the local file.
- `hasLocalModifications` no longer reports false positives for position-only `source:` differences, preventing unnecessary merge-mode activations.
- `MoveTopLevelFieldToEnd` is a single-pass reconstruction, avoiding the multiple YAML round-trips that would accumulate formatting drift.

#### Negative
- The `current` version of the file is silently repositioned during every merge, normalising `source:` to end of frontmatter even when the user did not touch it. This is a quiet content change outside the user's diff.
- `MergeWorkflowContent` now performs additional transformations (`MoveTopLevelFieldToEnd` + a second `NormalizeWhitespace` call), increasing the code path that must be reasoned about when debugging merge issues.

#### Neutral
- `MoveTopLevelFieldToEnd` is a new exported function in `frontmatter_editor.go`, adding to the editor's API surface area.
- The fix requires tests for both modified paths: `TestHasLocalModifications/source_field_in_different_position_-_not_a_modification` and `TestMergeWorkflowContent_SourceInMiddle`.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
