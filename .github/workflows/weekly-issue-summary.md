---
on:
  schedule:
    - cron: "0 15 * * 1"  # Weekly on Mondays at 3 PM UTC
  workflow_dispatch:
permissions:
  contents: read
  issues: read
  actions: read
engine: copilot
tools:
  github:
    allowed:
      - search_issues
      - get_issue
safe-outputs:
  create-issue:
    title-prefix: "[Weekly Summary] "
    labels: [automation, weekly-report]
---

# Weekly Issue Summary

Analyze all issues opened in the repository ${{ github.repository }} over the last 7 days.

Create a comprehensive summary that includes:
- Total number of issues opened
- List of issue titles with their numbers and authors
- Any notable patterns or trends (common labels, types of issues, etc.)

Format the summary clearly with proper markdown formatting for easy reading.
