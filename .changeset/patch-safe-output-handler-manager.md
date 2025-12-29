---
"gh-aw": patch
---

Implement safe output handler manager to consolidate multiple
safe-output steps into a single dispatch step.

This change introduces a JavaScript handler manager and updates
the Go compiler to emit one unified step for common safe output
types (create_issue, add_comment, create_discussion, close_issue,
close_discussion). Workflows were recompiled to use the new
manager.

Files changed: actions/setup/js/safe_output_handler_manager.cjs,
pkg/workflow/compiler_safe_outputs_core.go, and multiple workflow
lock files.

