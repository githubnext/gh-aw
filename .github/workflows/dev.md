---
on: 
  issue_comment:
    types: [created]
name: Dev
engine: codex
permissions:
  contents: read
  actions: read
safe-outputs:
  staged: true
  create-issue:
    title-prefix: "[dev] "
    labels: [dev, sub-task]
---

# Dev Workflow - Create Sub-Issue from Comment

When a user comments on an issue, analyze the comment and create a sub-issue with a summary.

## Context
- **Repository**: ${{ github.repository }}
- **Parent Issue**: #${{ github.event.issue.number }}
- **Comment**: "${{ needs.activation.outputs.text }}"

## Task

Analyze the comment and create a sub-issue that summarizes the main points or action items mentioned in the comment. Use the `parent` field to explicitly link the new issue as a sub-issue of #${{ github.event.issue.number }}.

The sub-issue should include:
- A clear title summarizing the comment
- A body with the key points or action items
- The parent issue number using the `parent` field

Example output format:
```json
{
  "type": "create-issue",
  "title": "Summary of comment discussion",
  "body": "Key points:\n- Point 1\n- Point 2\n\nOriginal comment by @user in #123",
  "parent": 123
}
```