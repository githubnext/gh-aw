---
"gh-aw": patch
---

Avoid regular-expression stack exhaustion when redacting runtime masks from multi-megabyte JSON and mixed-log artifacts, while preserving fail-closed cleanup for genuine redaction failures.

Keep malformed-token recovery linear, preserve unrelated plain-text escapes when masking encoded values, and match configured secrets together with runtime masks to prevent fragmentation or encoding changes from bypassing redaction.

Sanitize malformed ordinary JSON artifacts as whole-file text, matching raw and escaped credentials together even inside recoverable quoted spans or across line breaks.

Preserve bundled safe-output schema vocabulary during redaction so short masks do not corrupt review message fields, dispatch types, or collector configuration. Continue redacting payload data and fail closed on malformed executable records, unsafe custom routing changes, or conflicts with control data.

Coalesce overlapping built-in, configured, and runtime credentials in executable safe-output payloads without reparsing their already-decoded free-form strings.
