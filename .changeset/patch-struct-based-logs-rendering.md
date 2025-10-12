---
"gh-aw": patch
---

Refactor logs command to use struct-based console rendering system

Updated the logs command to use the same struct-based rendering approach as the audit command, improving code maintainability and consistency. All data structures now use unified types for both console and JSON output with proper struct tags.
