---
"gh-aw": patch
---

Fixed filtered bundle generation failing in repositories that configure checkout hooks (for example Git LFS). The temporary worktree used to apply the filtered patch invoked repository `post-checkout` and apply-patch hooks, so `git worktree add` failed when `git-lfs` was missing from the safe-outputs environment and the target branch was misreported as missing. These internal synthesis operations now run with an empty `core.hooksPath`.
