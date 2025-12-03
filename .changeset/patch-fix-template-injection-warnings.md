---
"gh-aw": patch
---

Fix template injection warnings by moving GitHub expressions into environment
variables and documenting safe cases.

Moved `needs.release.outputs.release_id` and `github.server_url` into `env` to
avoid template-injection scanner false positives while keeping behavior
unchanged. Documented that other flagged expressions use trusted GitHub
context and require no change.

Fixes: githubnext/gh-aw#5299

