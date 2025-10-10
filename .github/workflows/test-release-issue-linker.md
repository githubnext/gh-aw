---
on:
  issues:
    types: [opened]

permissions: read-all

engine: claude

tools:
  github:
    allowed: [get_issue, list_issues, search_issues]

safe-outputs:
  add-comment:
    max: 100
    target: "*"
    target-repo: "github/customer-feedback"
---

# Link Release Tracking Issues to Related Customer Feedback

Analyze the issue ${{ github.event.issue.number }} from `github/releases` and match it with customer feedback issues in `github/customer-feedback` using a comprehensive search and scoring approach.

## Analysis Steps

1. Get the issue details from `github/releases`
2. Extract key terms and context
3. Search for related issues in `github/customer-feedback`
4. Score and rank matches
5. Add a comment linking to the best matches

Let me know what I should analyze!