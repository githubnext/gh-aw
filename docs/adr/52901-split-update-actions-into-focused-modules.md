# ADR-52901: Split update_actions.go Into Focused Modules

**Date**: 2026-08-15
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

`pkg/cli/update_actions.go` had grown to 1,144 lines, mixing five unrelated concerns into a single file: dependency-injection/caching scaffolding, GitHub release resolution, cooldown enforcement, `actions-lock.json` orchestration, and Markdown `uses:` ref rewriting. Files of this size are hard to navigate, review, and test in isolation. The test file `update_actions_test.go` mirrored the problem at 1,256 lines. The lack of clear file-level boundaries made it difficult for reviewers to understand which functions belonged to which responsibility and for tests to target a single concern without loading the full module.

### Decision

We will split `pkg/cli/update_actions.go` into five focused files within the same `cli` package, each responsible for exactly one concern:

- `update_actions_deps.go` — DI/caching scaffolding (`actionUpdateDeps`, `newCachedActionUpdateDeps`, `defaultActionUpdateDeps`)
- `update_actions_release.go` — release and SHA resolution (GitHub API lookup, `git ls-remote` fallback, cooldown-aware version selection)
- `update_actions_lockfile.go` — `actions-lock.json` update orchestration (`UpdateActions`, `updateActions`)
- `update_actions_workflow_files.go` — workflow `.md` file walking and recompilation (`UpdateActionsInWorkflowFiles`)
- `update_actions_content_refs.go` — `uses:` and `skills:` ref rewriting inside Markdown content

All exported symbols (`UpdateActions`, `UpdateActionsInWorkflowFiles`) and their signatures are preserved; this is a pure file-organisation refactor with no behavioural changes.

### Alternatives Considered

#### Alternative 1: Keep the Monolithic File

Leave `update_actions.go` as-is and rely on code navigation tools (IDE symbol search, `grep`) to orient contributors. This avoids churn to the file tree and keeps all related code in one place. However, the file had already proved hard to review and test in isolation, and the problem would compound as new features are added.

#### Alternative 2: Extract a New Sub-Package (`pkg/cli/updateactions/`)

Move the logic into a dedicated sub-package with its own public API, giving each concern its own file within that package. This provides stronger encapsulation and would prevent other `cli` code from calling internal helpers directly. The trade-off is that it requires updating all import paths and exporting previously-internal helpers, adding significant refactoring scope beyond the stated goal of a pure file-diet refactor. Given that no new abstraction boundary was needed — the five concerns are already coherent within the `cli` package — a within-package split was chosen.

### Consequences

#### Positive
- Each file can be read and reviewed independently; a contributor inspecting cooldown behaviour reads only `update_actions_release.go` rather than scrolling past 1,000 unrelated lines.
- Tests are co-located with the source they exercise: `update_actions_release_test.go` tests release resolution, reducing cognitive load when adding or debugging test cases.
- File names serve as a lightweight table of contents for the `update-actions` feature.

#### Negative
- The `cli` package now contains more files, which can make directory listings less scannable for contributors unfamiliar with the split.
- Cross-cutting refactors (e.g., changing `actionUpdateDeps`) require touching multiple files rather than one, though this is mitigated by the small number of files and clear ownership boundaries.

#### Neutral
- No import changes are required by callers: all exported identifiers remain in `package cli`.
- The matching test split preserves all existing test cases without modification to test logic.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
