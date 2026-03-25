---
"gh-aw": patch
---

Added post-job cleanup for the `actions/setup` action to remove `/tmp/gh-aw/` after workflow execution, and updated checkout behavior so action runtime files are preserved for the post step.
