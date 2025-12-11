---
"gh-aw": patch
---

Introduce a typed `WorkflowStep` struct and helper methods for safer,
type-checked manipulation of GitHub Actions steps. Replace ad-hoc
`map[string]any` handling in step-related code with the new type where
possible, add conversion helpers, and add tests. Also fix
`ContinueOnError` to accept both boolean and string values.

Fixes githubnext/gh-aw#6053

