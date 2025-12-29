---
"gh-aw": patch
---

Add the safe output handler manager implementation and related refactorings.

This patch documents the internal refactor that centralizes safe-output message
dispatch into a single JavaScript handler manager and accompanying Go compiler
changes. The change simplifies workflows (multiple conditional steps → one
handler manager step), improves error handling, and keeps backward
compatibility with existing handlers. Includes tests and documentation.

