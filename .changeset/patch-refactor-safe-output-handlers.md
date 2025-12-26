---
"gh-aw": patch
---

Refactor safe output handlers into a sequential processing system using a
handler registry and explicit temporary ID map propagation.

This change extracts individual safe-output handlers (create_issue,
create_discussion, add_comment, update_issue, update_discussion,
close_issue, close_discussion) into their own files and consolidates the
processing loop into an ordered registry that tracks temporary ID map
availability via a shared context. This is an internal refactor that
improves maintainability and paves the way for migrating additional
handlers in the future.

