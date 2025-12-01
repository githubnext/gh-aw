---
on: 
  workflow_dispatch:
name: Dev
description: Find issues with "[deps]" in title and assign to Copilot agent
timeout-minutes: 5
strict: false
engine: copilot
permissions:
  contents: read
  issues: read
tools:
  github:
    toolsets: [repos, issues]
safe-outputs:
  assign-to-agent:
    name: copilot
---
# Dependency Issue Assignment

Find an open issue in this repository with "[deps]" in the title and assign it to the Copilot agent for resolution.

## Task

1. **Search for issues**: Use GitHub search to find open issues with "[deps]" in the title:
   ```
   is:issue is:open "[deps]" in:title repo:${{ github.repository }}
   ```

2. **Filter out assigned issues**: Skip any issues that already have Copilot as an assignee.

3. **Assign to Copilot**: For the first suitable issue found, use the `assign_to_agent` tool to assign it to the Copilot agent.

**Agent Output Format:**
```json
{
  "type": "assign_to_agent",
  "issue_number": <issue_number>,
  "agent": "copilot"
}
```

If no suitable issues are found, output a message indicating that no "[deps]" issues are available for assignment.