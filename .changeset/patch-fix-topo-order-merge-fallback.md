---
"gh-aw": patch
---

Fixed `push_signed_commits.cjs` to preserve commit replay order with `--topo-order` and fall back to `git push` when merge commits are detected, preventing incorrect signed-commit push behavior.
