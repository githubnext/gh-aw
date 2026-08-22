---
name: branch-for-pr
description: Create, switch to, and verify a real local branch (distinct from the base branch) before making PR-bound edits or calling the safe-outputs create_pull_request tool.
---

# Branch Before PR

Use this skill whenever a gh-aw workflow task requires opening a pull request for generated or edited files — for example, calling the safe-outputs `create_pull_request` tool, making changes described as being "on a named branch," or committing changes intended for review rather than a direct push to the base branch.

## Why this matters

`create_pull_request` needs a real local branch that (a) actually exists, (b) is different from the base branch, and (c) has at least one commit not present on the base branch. If an agent only *refers* to a branch name as a string — without actually running the git commands to create and switch to it — it can end up editing files while still on the base branch (commonly `main`). The PR call then fails with an error like "Branch 'main' equals base_branch 'main'. Cannot create a pull request from a branch onto itself," and any patch-generation fallback also fails (`git rev-parse --verify refs/heads/<branch>` → "fatal: Needed a single revision"; `git rev-list --count ...` → "fatal: ambiguous argument: unknown revision"), ultimately reporting "No changes to commit - no commits found." The same failure class also covers forgetting to switch branches mid-task, reusing a stale/wrong branch, or passing a `branch` parameter that doesn't match the agent's actual current git branch.

## Procedure

1. **Determine the base branch and the intended target/PR branch name** before editing anything (base is commonly `main`; the target branch name should be unique, e.g. include a task/topic identifier).
2. **Create and switch to the target branch before any file edits**: run `git checkout -b <unique-branch-name>`.
3. **Verify the branch switch succeeded**: run `git branch --show-current` and confirm the output exactly matches the intended branch name. If it doesn't, fix the branch state before proceeding — do not continue editing on the wrong branch.
4. **Only make file edits after verification succeeds.**
5. **Commit the changes** on that branch (`git add` / `git commit`).
6. **Re-verify immediately before calling `create_pull_request`:**
   - current branch (`git branch --show-current`) equals the intended target branch;
   - current branch is NOT equal to the base branch;
   - the local branch ref exists (e.g. `git rev-parse --verify refs/heads/<branch>` succeeds);
   - at least one commit/diff exists on the branch relative to the base branch (not "no changes to commit").
7. **Call `create_pull_request` with `branch` set to the exact verified current branch name** from `git branch --show-current` — never an invented or remembered string that was not actually checked out.

## Verification checklist

- [ ] Branch was created/checked out *before* any file edits were made
- [ ] `git branch --show-current` output matches the intended PR branch name exactly
- [ ] current branch != base branch
- [ ] the local branch ref exists (`git rev-parse --verify refs/heads/<branch>` succeeds)
- [ ] at least one commit or diff is attributable to the branch (not empty/no-changes)
- [ ] the `branch` parameter passed to `create_pull_request` exactly matches the verified current branch

## Stop conditions

- If you cannot create or switch to the intended branch, stop and report the git error rather than retrying `create_pull_request` with a branch name that was never actually checked out.
- If verification shows the current branch is the base branch or an unexpected branch, fix the branch state first — do not call `create_pull_request` in this state.
- If the branch parameter you're about to pass differs from `git branch --show-current`, resolve the mismatch before calling the tool.
- If no commits/diff exist for the branch, do not call `create_pull_request` — there is nothing to submit; consider `noop` or `report_incomplete` instead, per the workflow's safe-output conventions.
