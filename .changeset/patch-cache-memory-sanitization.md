---
"gh-aw": patch
---

Add cache-memory artifact sanitization to threat detection jobs.

- Detection jobs download and analyze `cache-memory` artifacts for secrets and sensitive data.
- Adds `{CACHE_MEMORY_FILES}` placeholder and `CACHE_MEMORY_DIRS` env var to threat detection setup.
- Introduces helper functions for consistent cache path handling and backward compatibility.

Fixes githubnext/gh-aw#5437

