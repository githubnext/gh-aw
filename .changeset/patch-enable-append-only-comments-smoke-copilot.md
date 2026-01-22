---
"gh-aw": patch
---

Enable append-only comments for the `smoke-copilot` workflow.

The workflow now posts new status comments for each run instead of editing
the original activation comment. This adds `append-only-comments: true`
to the messages configuration so timeline updates create discrete comments.

Files changed: schema and `.github/workflows/smoke-copilot.md` (compiled lock updated).

