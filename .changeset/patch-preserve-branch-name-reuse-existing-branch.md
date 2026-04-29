---
"gh-aw": patch
---

Fix `create-pull-request` to reuse an existing remote branch when `preserve-branch-name: true` is enabled. Previously, the handler refused to push if the named branch already existed on the remote, blocking workflows that intentionally maintain long-lived reusable branches across iterations (e.g. autoloop programs whose previous PR was merged but whose branch still exists). The handler now force-deletes the stale remote ref and recreates the branch from the agent's local HEAD, matching the intent of `preserve-branch-name`.
