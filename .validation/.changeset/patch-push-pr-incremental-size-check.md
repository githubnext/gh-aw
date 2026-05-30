---
"gh-aw": patch
---

Fixed `push_to_pull_request_branch` `max_patch_size` check incorrectly measuring the patch from the default branch (checkout base) instead of from the existing PR branch head, causing every push to fail on long-running branches that accumulate iterations.

The `max_patch_size` check now uses the **incremental net diff** between `origin/<branch>` and the new working state, not the size of the format-patch transport file or the cumulative diff from the default branch. This means the limit reflects how much the branch will actually change in the push, not the cumulative divergence of the long-running branch from `main`.

Changes:
- `generate_git_patch.cjs`: in incremental mode, also computes and returns `diffSize` (size of `git diff <baseRef>..<branch>`), and refuses to fall through to checkout-base strategies (`GITHUB_SHA..HEAD` / merge-base with default branch) if Strategy 1 fails.
- `safe_outputs_handlers.cjs`: passes the incremental `diffSize` to the safe-output entry as `diff_size`.
- `push_to_pull_request_branch.cjs`: prefers `message.diff_size` for the `max_patch_size` check, falling back to the patch file size when the field is absent (backward compatible).
