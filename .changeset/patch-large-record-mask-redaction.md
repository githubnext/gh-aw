---
"gh-aw": patch
---

Avoid regular-expression stack exhaustion when redacting runtime masks from multi-megabyte JSON and mixed-log artifacts, while preserving fail-closed cleanup for genuine redaction failures.

Preserve bundled safe-output schema vocabulary during redaction so short masks do not corrupt review message fields, dispatch types, or collector configuration. Continue redacting payload data and fail closed on malformed executable records, unsafe custom routing changes, or conflicts with control data.
