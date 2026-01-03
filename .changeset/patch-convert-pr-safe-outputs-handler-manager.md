---
"gh-aw": patch
---

Convert PR-related safe outputs and the `hide-comment` safe output to the
handler-manager architecture used by other safe outputs (e.g. `create-issue`).

This is an internal refactor: handlers now use the handler factory pattern,
enforce max counts, return result objects, and are managed by the handler
manager. TypeScript, linting, and Go formatting were applied.

