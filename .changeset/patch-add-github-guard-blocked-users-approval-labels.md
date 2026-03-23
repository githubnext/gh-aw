---
"gh-aw": patch
---

Add `blocked-users` and `approval-labels` support to `tools.github` guard policies, including schema/parser/validation updates and runtime parsing via `parse_guard_list.sh` — which merges compile-time static values with `GH_AW_GITHUB_BLOCKED_USERS` and `GH_AW_GITHUB_APPROVAL_LABELS` org/repo variables into proper JSON arrays (split on comma/newline, validated, jq-encoded).
