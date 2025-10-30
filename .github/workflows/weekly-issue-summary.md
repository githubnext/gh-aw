---
on:
  schedule:
    - cron: "0 15 * * 1"  # Weekly on Mondays at 3 PM UTC
  workflow_dispatch:
permissions:
  issues: read
engine: copilot
network:
  firewall: true
tools:
  edit:
  bash:
    - "*"
  github:
    toolsets: 
      - issues
    allowed:
      - search_issues
      - issue_read
safe-outputs:
  create-discussion:
    title-prefix: "[Weekly Summary] "
    category: "Audits"
imports:
  - shared/reporting.md
---

# Weekly Issue Summary

Analyze all issues opened in the repository ${{ github.repository }} over the last 7 days.

Create a comprehensive summary that includes:
- Total number of issues opened
- List of issue titles with their numbers and authors
- Any notable patterns or trends (common labels, types of issues, etc.)

Format the summary clearly with proper markdown formatting for easy reading.
