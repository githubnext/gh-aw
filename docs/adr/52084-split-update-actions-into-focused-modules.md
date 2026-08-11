# ADR-52084: Split update-actions Implementation into Focused Modules

**Date**: 2026-08-11
**Status**: Draft
**Deciders**: Unknown

---

### Context

`pkg/cli/update_actions.go` grew to 1,144 lines, exceeding the repository's 800-line file-size threshold and becoming the largest non-test Go source file in the codebase. The file mixed four loosely related responsibilities in a single compilation unit: action classification (`isCoreAction`, `isGhAwNativeAction`), release version resolution via GitHub API and git fallback (`getLatestActionRelease`, `getLatestActionReleaseViaGit`, cooldown gating), lockfile orchestration (`UpdateActions`, `updateActions`), and workflow Markdown reference rewriting (`UpdateActionsInWorkflowFiles`, `updateSkillRefsInContent`, `updateActionRefsInContentWithDeps`). This coupling made it hard to locate, read, and test individual concerns without scrolling through the entire file.

### Decision

We will split `pkg/cli/update_actions.go` into four focused files within the same `cli` package, preserving all public API signatures (`UpdateActions`, `UpdateActionsInWorkflowFiles`) and all existing behaviour unchanged:

1. **`update_actions_deps.go`** — dependency-injection struct (`actionUpdateDeps`), cache types, `newCachedActionUpdateDeps`, and `defaultActionUpdateDeps`.
2. **`update_actions_release.go`** — action classification helpers, GitHub Releases API resolution, git-tag fallback resolution, SHA peeling, and cooldown-window gating.
3. **`update_actions.go`** (reduced) — top-level `UpdateActions` entry point and `updateActions` lockfile orchestration.
4. **`update_actions_workflow_refs.go`** — `UpdateActionsInWorkflowFiles` entry point, workflow-file walking, `uses:` action-ref rewriting, and `source:` skill-ref rewriting.

The split follows domain boundaries already visible in the file, requires no new interfaces or packages, and leaves callers in `update_command.go` and test files unmodified.

### Alternatives Considered

#### Alternative 1: Keep the monolith

Leave `update_actions.go` as a single file. This requires zero refactoring effort and zero risk of regressions, but the file will continue to grow as new concerns are added. The four distinct domains will remain entangled, making code review and targeted testing progressively harder. The repository's file-size threshold would be permanently violated.

#### Alternative 2: Extract to a dedicated package (`pkg/updateactions`)

Move all action-update logic to a new top-level package instead of splitting within `pkg/cli`. This would create a cleaner architectural boundary and make the separation visible to package consumers. However, it requires updating import paths in multiple callers and test files, changing the exported symbol locations, and performing a larger structural change — all without any behaviour difference. Given that no external package currently imports these symbols directly (they are CLI-internal), the added complexity is not justified at this stage.

### Consequences

#### Positive
- Each file encapsulates a single concern; a developer working on cooldown logic only needs to open `update_actions_release.go` rather than scrolling a 1,144-line file.
- Test files can be named and organised to mirror the new source boundaries (`update_actions_release_test.go`, `update_actions_workflow_refs_test.go`), making the relationship between source and test explicit.
- The repository's file-size threshold is satisfied for all resulting files (each is under 500 lines).

#### Negative
- Understanding the complete action-update flow now requires opening and cross-referencing multiple files instead of reading one.
- The refactor produces no user-visible functionality and adds commit churn that reviewers must verify as purely mechanical.

#### Neutral
- The public API (`UpdateActions`, `UpdateActionsInWorkflowFiles`) and all function signatures remain unchanged, so no callers require modification.
- `actionUpdateDeps` continues to act as the shared dependency-injection seam between resolution and rewriting logic; no new interfaces are introduced.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
