---
"gh-aw": patch
---

Apply struct-based rendering to status command

Refactored the `status` command to use the struct tag-based console rendering system, following the guidelines in `.github/instructions/console-rendering.instructions.md`. The change reduces code duplication by eliminating manual table construction and improves maintainability by defining column headers once in struct tags. JSON output continues to work exactly as before.
