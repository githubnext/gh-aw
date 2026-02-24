---
on: weekly
permissions:
  contents: read
  issues: read
  pull-requests: read
tools:
  github:
safe-outputs:
  create-issue:
    title-prefix: '[dependabot-burner] '
---
# Dependabot Burner

- Find all open Dependabot PRs.
- Create bundle issues, each for exactly **one runtime + one manifest file**.

**Important**: If no action is needed after completing your analysis, you **MUST** call the `noop` safe-output tool with a brief explanation. Failing to call any safe-output tool is the most common cause of safe-output workflow failures.

```json
{"noop": {"message": "No action needed: [brief explanation of what was analyzed and why]"}}
```
