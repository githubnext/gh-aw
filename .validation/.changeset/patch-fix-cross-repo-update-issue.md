---
"gh-aw": patch
---

Fixed cross-repo `update-issue`/`update_pull_request` safe-outputs by honoring `target-repo` even without an explicit `repo` and allowing `target-repo: "*"` to validate other repos.
