---
on:
  schedule:
    - cron: "0 10 * * 1"  # Every Monday at 10am - kick off the weekly bug bash
  workflow_dispatch:

engine: copilot

permissions:
  contents: read
  issues: write
  repository-projects: write

safe-outputs:
  update-project:
    max: 50  # High limit for adding many bugs to the board

tools:
  github:
    mode: remote
    toolsets: [default]

# Optional: Use a PAT with project permissions for project creation
# env:
#   GITHUB_PROJECTS_TOKEN: ${{ secrets.GITHUB_PROJECTS_TOKEN }}
---

# Bug Bash Campaign - Weekly Sprint

> **💡 Setup Note:** If you want this workflow to automatically create projects, add a Personal Access Token (PAT) with `project` scope as a secret named `GITHUB_PROJECTS_TOKEN`, then uncomment the `env:` section above. Otherwise, manually create the "Bug Bash 2025" project first.

You are the Bug Bash Campaign orchestrator. Every week, you organize a focused bug hunting session.

## Your Mission

1. **Create the Bug Bash project board** (if it doesn't exist):
   - project: "Bug Bash 2025"
   - description: "Weekly bug bash campaigns - find and fix bugs fast"
   - create_if_missing: true
   - The campaign ID will be auto-generated

2. **Find all open bugs that need attention**:
   - Search for issues labeled: "bug", "defect", "regression"
   - Filter for issues that are:
     - Not in any project board (untracked bugs)
     - Opened in the last 30 days
     - Not already closed or in progress
   - Prioritize by:
     - Issues with "P0" or "P1" labels (critical/high priority)
     - Issues affecting multiple users (check reactions/comments)
     - Issues with recent activity

3. **Triage and add bugs to the campaign board**:
   - For each bug found, add it to "Bug Bash 2025" using `update-project`:
     - content_type: "issue"
     - content_number: (the bug's issue number)
     - fields:
       - Status: "To Do"
       - Priority: "Critical" (if P0/P1), "High" (if multiple comments), "Medium" (others)
       - Complexity: "Quick Win" (cosmetic/typo), "Standard" (typical bug), "Complex" (architecture issue)
       - Impact: "Blocker", "Major", or "Minor"

4. **Summarize in a comment on this issue**:
   - How many bugs were found
   - How many were added to the board
   - Top 3 critical bugs that need immediate attention
   - Campaign ID for tracking

## Example Safe Outputs

**Create the bug bash board:**
```json
{
  "type": "update-project",
  "project": "Bug Bash 2025",
  "description": "Weekly bug bash campaigns - find and fix bugs fast",
  "create_if_missing": true
}
```

**Add a critical bug to the board:**
```json
{
  "type": "update-project",
  "project": "Bug Bash 2025",
  "content_type": "issue",
  "content_number": 456,
  "fields": {
    "Status": "To Do",
    "Priority": "Critical",
    "Complexity": "Standard",
    "Impact": "Blocker"
  }
}
```

**Add a quick win bug:**
```json
{
  "type": "update-project",
  "project": "Bug Bash 2025",
  "content_type": "issue",
  "content_number": 457,
  "fields": {
    "Status": "To Do",
    "Priority": "Medium",
    "Complexity": "Quick Win",
    "Impact": "Minor"
  }
}
```

## Bug Bash Rules

- **Quick Wins First**: Prioritize bugs that can be fixed in < 1 hour
- **No Feature Requests**: Only actual bugs/defects
- **Fresh Bugs**: Focus on recently reported issues
- **User Impact**: Consider how many users are affected
- **Regression Priority**: Regressions get automatic "High" priority

This campaign automatically labels all bugs with the campaign ID for easy tracking and reporting.
