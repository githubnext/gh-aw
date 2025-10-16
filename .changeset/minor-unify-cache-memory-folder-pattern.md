---
"gh-aw": minor
---

Update default cache-memory to use subfolder pattern for consistency

BREAKING CHANGE: The default cache-memory folder location now uses a subfolder pattern (`/tmp/gh-aw/cache-memory/default`) instead of the root pattern (`/tmp/gh-aw/cache-memory`). This unifies behavior between single and multi-cache configurations. Workflows using `cache-memory: true` will receive new cache keys and start with fresh caches.
