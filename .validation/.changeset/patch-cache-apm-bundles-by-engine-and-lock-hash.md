---
"gh-aw": patch
---

Cache APM bundles by `engine_id` and workflow lock file hash so runs can reuse previously packed bundles across workflow executions while still uploading an artifact for reliable same-run restore fallback.

Expose `engine_id` from the activation job outputs so shared workflows can build engine-specific cache keys when preparing agent dependencies.
