---
"gh-aw": patch
---

Wrapped unguarded synchronous filesystem calls in `actions/setup/js/` with `try/catch` so I/O failures no longer crash the action.
