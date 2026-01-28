---
timeout-minutes: 10
strict: true
on:
  schedule: daily
  workflow_dispatch:
permissions:
  issues: read
  pull-requests: read
  contents: read
engine: copilot
tools:
  github:
    toolsets: [issues, pull_requests, repos]
safe-outputs:
  assign-to-user:
    target: "*"
  add-comment:
    target: "*"
---

{{#runtime-import? .github/shared-instructions.md}}

# Auto-Assign Issue

Find ONE open issue that:
- **Has no assignees** - This is CRITICAL. You MUST verify this in two places:
  1. Use `no:assignee` in your search query
  2. After retrieving the issue details, explicitly check that `issue.assignees` is an empty array or has length 0
  3. **SKIP AND EXIT if `issue.assignees` is not empty** - DO NOT assign issues that already have assignees
- Does not have label `ai-generated`
- Does not have a `campaign:*` label (these are managed by campaign orchestrators)
- Does not have labels: `no-bot`, `no-campaign`
- Was not opened by `github-actions` or any bot

**Verification Steps (REQUIRED):**
1. Search for issues using `no:assignee` filter
2. Pick the oldest unassigned issue from search results
3. Retrieve full issue details using `issue_read`
4. **VERIFY** that `issue.assignees` is empty (length === 0)
5. **IF assignees is NOT empty, EXIT WITHOUT ASSIGNING** - the issue was assigned by someone else
6. Only if `issue.assignees` is empty, proceed to list contributors and assign

Then list the 5 most recent contributors from merged PRs. Pick one who seems relevant based on the issue type.

If you find a match AND the issue has no assignees:
1. Use `assign-to-user` to assign the issue
2. Use `add-comment` with a short explanation (1-2 sentences)

If no unassigned issue exists, exit successfully without taking action.
