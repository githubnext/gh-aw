---
"gh-aw": patch
---

Remove workflow cancellation API calls from compiler

The compiler no longer uses the GitHub Actions cancellation API. Workflow cancellation is now handled through job dependencies and `if` conditions, resulting in a cleaner architecture. This removes the need for `actions: write` permission in the `add_reaction` job and eliminates 125 lines of legacy code.
