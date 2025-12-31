---
"gh-aw": patch
---

Warn when safe output messages are skipped due to missing handlers

When safe output messages (for example, `create_issue`) are sent but no
handler is loaded, they were silently skipped with only a debug log.
This change converts those debug logs to warnings, records skipped
messages in the processing results, and improves the processing summary
to separately report skipped vs failed messages.

