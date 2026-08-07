---
"gh-aw": patch
---

Make repo-memory glob mismatches visible instead of silent. `push_repo_memory` now emits a warning listing the rejected paths when every file in the memory artifact is filtered out by `file-glob`, and the generated repo-memory prompt documents that slashless patterns (e.g. `*.md`) only match files placed exactly one folder deep in the memory folder.
