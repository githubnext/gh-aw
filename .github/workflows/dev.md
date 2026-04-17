---
on:
  workflow_dispatch:
  label_command: dev
  schedule:
    - cron: 'daily around 9:00'  # ~9 AM UTC
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
features:
  mcp-cli: true
  copilot-requests: true

tools:
  mount-as-clis: true
---

# Daily Status Report

Generate a daily status report for the gh-aw project, focusing on documentation quality.

**Requirements:**

1. **Find documentation problems reported in issues**: Search GitHub issues for mentions of documentation bugs, unclear instructions, missing documentation, or incorrect documentation. Look for patterns like "docs", "documentation", "unclear", "wrong", "missing", "broken", "outdated".

2. **Cross-reference with current documentation**: For each documentation problem found in issues, search the repository documentation to find the relevant section that the issue is referencing or that could answer the question raised.

3. **Compile a report** summarizing:
   - Issues that report documentation problems (with issue numbers and titles)
   - The corresponding documentation sections that may need updating
   - Any issues where the documentation actually already contains the answer (and the issue could be closed with a pointer)
   - Gaps where no documentation exists for a reported problem

4. Post the report as an issue with the date in the title.

Keep the report informative but concise.

**Important**: If no action is needed after completing your analysis, you **MUST** call the `noop` safe-output tool with a brief explanation. Failing to call any safe-output tool is the most common cause of safe-output workflow failures.

```json
{"noop": {"message": "No action needed: [brief explanation of what was analyzed and why]"}}
```
