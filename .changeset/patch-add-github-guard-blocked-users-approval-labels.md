---
"gh-aw": patch
---

Add `blocked-users` and `approval-labels` support to `tools.github` guard policies, including schema/parser/validation updates and safe runtime fallback merging with `GH_AW_GITHUB_BLOCKED_USERS` and `GH_AW_GITHUB_APPROVAL_LABELS` via `toJSON()` rendering.
