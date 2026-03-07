---
"gh-aw": patch
---
Silence expected git fetch/pull failures in `push_repo_memory` by routing their output to `core.debug`, keeping error annotations out of Action logs.
