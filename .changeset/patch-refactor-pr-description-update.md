---
"gh-aw": patch
---

Refactor PR description updates: extract helper module, add `replace-island` mode for
idempotent PR description sections, make footer messages customizable via workflow
frontmatter, and add tests.

This change introduces a new helper `update_pr_description_helpers.cjs`, a
`replace-island` operation mode that updates workflow-run-scoped islands in PR
descriptions, and customizable footer messages via `messages.footer` in the
workflow frontmatter. Tests were added for the helper and integration scenarios.

