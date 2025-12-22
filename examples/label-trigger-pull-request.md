---
name: Label Trigger Example - Pull Request
description: Example workflow demonstrating the pull_request labeled trigger shorthand syntax
on: pull_request labeled needs-review approved ready-to-merge
engine:
  id: codex
  model: gpt-5-mini
strict: true
---

# Label Trigger Example - Pull Request

This workflow demonstrates the pull_request labeled trigger shorthand syntax:

```yaml
on: pull_request labeled needs-review approved ready-to-merge
```

This short syntax automatically expands to:
- Pull request labeled with any of the specified labels
- Workflow dispatch trigger with an item_number input parameter

## Task

When this workflow is triggered, acknowledge that it was triggered and provide a brief summary.

Example output:
```
✅ Workflow triggered for PR!
📋 This workflow responds to label events on pull requests
```
