---
"gh-aw": patch
---

Add step summaries for safe-output processing results.

Safe-output handlers now generate collapsible step summaries for each processed
message, providing visibility into what was created or updated during workflow
execution. Body previews are truncated at 500 characters to avoid bloat. The
feature is implemented for both regular safe-outputs and project-based
safe-outputs via a shared helper module.

