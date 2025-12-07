---
"gh-aw": patch
---

Defer saving cache-memory artifacts until after threat detection validates agent output.

Agent jobs now upload cache-memory artifacts instead of saving cache directly when threat
detection is enabled. A new `update_cache_memory` job downloads the artifacts and saves
them to the GitHub Actions cache only after detection passes, preventing unvalidated
agent outputs from being persisted.

This change is a security-focused bugfix and internal behavior change (no public API
breaks).

