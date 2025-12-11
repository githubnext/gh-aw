---
"gh-aw": patch
---

Extract duplicate REST API wrappers into a shared factory `createEntityCallbacks` used by `close_issue.cjs` and `close_pull_request.cjs`.

This refactor reduces duplicated code, centralizes REST request wiring, and includes tests for both entity types. No user-facing API changes.

