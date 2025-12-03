---
on: 
  workflow_dispatch:
name: Dev
description: Find an open issue and assign it to mrjf
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
# Issue Assignment

Find an open issue in this repository and assign it to mrjf for resolution.

## Task

1. **Search for issues**: Use GitHub search to find open issues in this repository:
   ```
   is:issue is:open repo:${{ github.repository }}
   ```

2. **Filter out assigned issues**: Skip any issues that already have mrjf as an assignee.

3. **Pick an issue**: Select the first suitable unassigned issue found.

4. **Assign to mrjf**: Use the `assign_to_user` tool to assign the selected issue to mrjf.

**Agent Output Format:**
```json
{
  "type": "assign_to_user",
  "issue_number": <issue_number>,
  "assignee": "mrjf"
}
```

If no suitable issues are found, output a noop message indicating that no unassigned issues are available.