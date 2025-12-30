---
"gh-aw": patch
---

Migrate safe output handlers to a centralized handler config object and remove handler-specific environment variables.

All eight safe-output handlers (create_issue, add_comment, create_discussion, close_issue, close_discussion, add_labels, update_issue, update_discussion) were refactored to accept a single handler config object instead of reading many individual environment variables. This reduces the number of handler-specific env vars from 30+ down to 3 global env vars and centralizes configuration for easier testing and maintenance.

Files changed: multiple JavaScript safe output handlers under `actions/setup/js/` and related Go compiler cleanup in `pkg/workflow/`.

Benefits: explicit data flow, fewer environment variables, testable handlers, and simpler configuration.

