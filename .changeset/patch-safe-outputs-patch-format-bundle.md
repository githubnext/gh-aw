---
"gh-aw": patch
---

Add `patch-format: bundle` option to `create-pull-request` and `push-to-pull-request-branch` safe outputs. Set `patch-format: bundle` to transport changes via `git bundle` instead of `git format-patch`/`git am`, preserving merge commit topology, per-commit authorship and messages, and merge-resolution-only content. The default (`patch-format: am`) is unchanged.
