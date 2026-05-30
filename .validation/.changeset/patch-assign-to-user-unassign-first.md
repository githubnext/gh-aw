---
"gh-aw": patch
---

Add `unassign-first` configuration option to `assign-to-user` safe output. When enabled, automatically unassigns all current assignees from an issue or pull request before assigning new ones, solving the issue where GitHub's addAssignees API does not replace existing assignees.
