# ADR-43405: Dynamic Default-Branch Detection for `gh aw trial --host-repo`

**Date**: 2026-07-05
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

`gh aw trial --host-repo=.` fails immediately when the host repository's default branch is not `main`. The two internal helpers `commitAndPushWorkflow` and `copyTrialResultsToHostRepo` unconditionally ran `git pull origin main` and `git push origin main`, which do not exist in repositories that use `master`, `trunk`, or any other default branch name. Additionally, neither helper set `cmd.Dir` on the `git pull` command, which was a pre-existing bug that could silently operate on the wrong directory. The tool needs to work correctly across all repositories regardless of their default branch name.

### Decision

We will introduce `getCurrentBranchIn(dir string)` — a directory-aware variant of the existing `getCurrentBranch()` helper — and use it in both `commitAndPushWorkflow` and `copyTrialResultsToHostRepo` to read the actual default branch from the locally-checked-out clone (`tempDir`) at runtime. When detection fails (non-git directory, detached HEAD, command error), the implementation falls back to `"main"` with debug logging. Since `tempDir` is always a fresh clone of the host repo, `HEAD` reliably reflects the remote default branch.

### Alternatives Considered

#### Alternative 1: Explicit `--default-branch` CLI flag

Users could pass the default branch name as a flag to `gh aw trial`. This avoids any dynamic detection and makes the branch explicit.

This was not chosen because it places an unnecessary burden on every user to know their repo's default branch name. The local clone already carries this information, making auto-detection both simpler and more reliable. A flag would also break automation scripts that work across repos with different default branches.

#### Alternative 2: Query the remote for the default branch

The default branch could be queried from the remote via `git remote show origin` (parsing `HEAD branch:`) or the GitHub API. This would work regardless of local checkout state.

This was not chosen because `tempDir` is always a fresh clone of the host repo with a checked-out `HEAD`, so reading `HEAD` locally is equivalent to reading the remote default branch — while being faster, simpler, and not requiring network access or API credentials at that point in the workflow.

#### Alternative 3: Read default branch from GitHub API / repository metadata

The `gh` CLI or GitHub API could provide the default branch before cloning. This is authoritative but adds an API call and couples the implementation to the GitHub API.

This was not chosen for the same reason as Alternative 2: the local clone already reflects the remote's `HEAD`, and the added complexity is not justified for a simple bug fix.

### Consequences

#### Positive
- `gh aw trial --host-repo=.` now works correctly for repositories using `master`, `trunk`, or any other default branch name.
- Reuses the existing `getCurrentBranch()` infrastructure, keeping the codebase DRY; the new `getCurrentBranchIn(dir)` function is the single source of truth for branch detection with or without a directory argument.
- Adds comprehensive test coverage (`TestGetCurrentBranchIn`) for `main`, `master`, custom branch names, and the non-git-directory error path.
- Fixes the pre-existing missing `cmd.Dir` on `git pull` calls, preventing potential silent operation on the wrong working directory.

#### Negative
- If `tempDir` is somehow in detached HEAD state (edge case: shallow clone, CI runner behavior), the fallback to `"main"` will be used silently (only debug-logged), which could cause a push failure or push to an unintended branch without a user-visible error message.
- Relies on local checkout state (`HEAD`) rather than an authoritative remote query; if the remote default branch changes between clone and push (extremely unlikely in normal usage), the push could target a stale branch name.

#### Neutral
- The `getCurrentBranch()` function (no-dir variant) is refactored to delegate to `getCurrentBranchIn("")`, maintaining backward compatibility with all existing callers.
- The fallback branch name remains `"main"` for compatibility with the majority of GitHub repositories.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
