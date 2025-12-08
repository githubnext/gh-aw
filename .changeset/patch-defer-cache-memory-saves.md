---
"gh-aw": patch
---

Defer cache-memory saves until after threat detection validates agent output.

The agent job now uploads cache-memory artifacts and the new `update_cache_memory`
job saves those artifacts to the Actions cache only after threat detection passes.

This fixes a race where cache memories could be saved before detection validated
the agent's output.

Fixes githubnext/gh-aw#5763

