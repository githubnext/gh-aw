---
"gh-aw": patch
---

Optimize `push_to_pull_request_branch` bundle transport in incremental mode by excluding `origin/<base_branch>` objects when available. This keeps merge-conflict-resolution workflows from re-sending upstream base-branch history that the remote already has, reducing bundle size while preserving the same commit tip.
