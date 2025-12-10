---
"gh-aw": patch
---

Add a new `bots` frontmatter field that allows listing GitHub Apps/bots authorized to trigger a workflow.

This change documents the schema and implementation work: schema update, Go parsing and env passing, JavaScript validation, and tests.
