---
"gh-aw": patch
---

Improve conclusion job step handling by merging no-op processing into a single step, removing a dead `handle_create_pr_error` step, and reusing serialized safe output message config in conclusion env generation.
