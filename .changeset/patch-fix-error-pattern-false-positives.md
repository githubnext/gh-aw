---
"gh-aw": patch
---

Fix error pattern false positives in workflow validation

The error validation step was incorrectly flagging false positives when workflow output contained filenames or text with "error" as a substring. Updated error patterns across all AI engines (Copilot, Claude, and Codex) to use word boundaries (`\berror\b`) instead of matching any occurrence of "error", ensuring validation correctly distinguishes between actual error messages and informational text.
