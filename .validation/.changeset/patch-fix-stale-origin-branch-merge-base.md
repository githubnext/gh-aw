---
"gh-aw": patch
---

Fixed `create_pull_request` patch (full mode) using a stale `origin/<branchName>` as the base ref when that remote-tracking ref existed locally, causing the patch to include commits the agent never made.

In `generate_git_patch.cjs` full mode, the code previously short-circuited to `origin/<branchName>` whenever that ref existed locally. But `origin/<branchName>` is fetched at workflow startup and can be arbitrarily stale — typically pointing at where the remote branch was before the agent run, not the state of the branch *before the agent made changes*. When the local branch is fast-forwarded to the default branch during the agent run (a common pattern for long-running iterative workflows), the resulting patch erroneously included every commit between the old branch tip and the current `main`.

Full mode now always computes the merge-base with the default branch, matching the behavior of the previous fallback path. Incremental mode (`push_to_pull_request_branch`) is unchanged — its use of `origin/<branchName>` as a base is intentional and correct for that workflow.
