---
"gh-aw": patch
---

Fix `create-pull-request` and `push-to-pull-request-branch` safe outputs diverging from the agent job's on-disk checkout layout when `target-repo` named a specific repository (not `"*"`). The `safe_outputs` job now mirrors the agent layout whenever any `checkout:` entry is checked out into a subdirectory: every repository is checked out to the same path the agent used (workflow repo at the root plus each cross-repo checkout at its `path:`), instead of collapsing to a single checkout of the target at the workspace root. Subdirectory cross-repo checkouts also emit the same non-empty blob filter and partial-clone-marker reset as the agent job, fixing intermittent `couldn't find remote ref` failures during the post-checkout fetch.
