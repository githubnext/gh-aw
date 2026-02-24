---
on:
  workflow_dispatch:
  schedule:
    - cron: '0 9 * * *'  # Daily at 9 AM UTC
name: Dev
description: Daily status report for gh-aw project
timeout-minutes: 30
strict: false
engine: copilot

permissions:
  contents: read
  issues: read
  pull-requests: read

safe-outputs:
  create-issue:
    expires: 7d
    title-prefix: "[Daily Report] "
---

# Daily Status Report

Generate a daily status report for the gh-aw project.

**Requirements:**
1. Analyze the current state of the repository
2. Check for recent commits, pull requests, and issues
3. Identify any potential issues or areas needing attention
4. Create a comprehensive daily status report
5. Post the report as an issue with the date in the title

Keep the report informative but concise.

**Important**: If no action is needed after completing your analysis, you **MUST** call the `noop` safe-output tool with a brief explanation. Failing to call any safe-output tool is the most common cause of safe-output workflow failures.

```json
{"noop": {"message": "No action needed: [brief explanation of what was analyzed and why]"}}
```
