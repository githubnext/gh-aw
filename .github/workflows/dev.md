---
on:
  workflow_dispatch:
name: Dev
description: Find a random issue suitable for Copilot and assign it to the agent
timeout-minutes: 5
strict: false
engine: claude
permissions:
  contents: read
  issues: read
tools:
  github: true
safe-outputs:
  assign-to-agent:
    name: copilot
---
# Find and Assign Issue to Copilot

Find a random open issue in this repository that would be suitable for GitHub Copilot to work on, then assign it to the Copilot agent.

## Task

1. **Find Suitable Issues**: Search for open issues in the repository that would be good candidates for Copilot to work on. Look for issues that:
   - Are well-defined with clear requirements
   - Involve code changes (bug fixes, features, improvements)
   - Have enough context and information
   - Are not too complex or vague
   - Don't require extensive human judgment or product decisions

2. **Select a Random Issue**: From the suitable candidates, pick one at random.

3. **Assign to Copilot**: Use the `assign_to_agent` safe output to assign the selected issue to the Copilot coding agent.

## Output Format

Use the assign_to_agent tool with the following format:

```json
{
  "type": "assign_to_agent",
  "issue_number": <issue_number>,
  "agent": "copilot"
}
```