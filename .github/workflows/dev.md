---
on: 
  workflow_dispatch:
name: Dev
description: Find issues with "[deps]" in title and assign to mrjf
timeout-minutes: 5
strict: false
engine: claude
permissions:
  contents: read
  issues: write
tools:
  github:
    toolsets: [repos, issues]
safe-outputs:
  assign-to-user:
    allowed: [mrjf]
    target: "*"
---
# Dependency Issue Assignment

Find an open issue in this repository with "[deps]" in the title and assign it to mrjf for resolution.

## Task

1. **Search for issues**: Use GitHub search to find open issues with "[deps]" in the title:
   ```
   is:issue is:open "[deps]" in:title repo:${{ github.repository }}
   ```

2. **Filter out assigned issues**: Skip any issues that already have mrjf as an assignee.

3. **Assign to mrjf**: For the first suitable issue found, use the `assign_to_user` tool to assign it to mrjf.

**Agent Output Format:**
```json
{
  "type": "assign_to_user",
  "issue_number": <issue_number>,
  "assignee": "mrjf"
}
```

If no suitable issues are found, output a message indicating that no "[deps]" issues are available for assignment.