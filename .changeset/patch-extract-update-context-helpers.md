---
"gh-aw": patch
---

Refactor: Extracted shared context validation helpers into
`update_context_helpers.cjs` and updated `update_issue.cjs` and
`update_pull_request.cjs` to import and use the shared helpers. This reduces
duplication and improves maintainability. Fixes githubnext/gh-aw#6563

