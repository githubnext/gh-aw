---
"gh-aw": patch
---

Avoid regular-expression stack exhaustion when redacting runtime masks from multi-megabyte JSON and mixed-log artifacts, while preserving fail-closed cleanup for genuine redaction failures.
