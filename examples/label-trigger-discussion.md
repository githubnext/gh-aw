---
name: Label Trigger Example - Discussion
description: Example workflow demonstrating the discussion labeled trigger shorthand syntax
on: discussion labeled question announcement help-wanted
engine:
  id: codex
  model: gpt-5-mini
strict: true
---

# Label Trigger Example - Discussion

This workflow demonstrates the discussion labeled trigger shorthand syntax:

```yaml
on: discussion labeled question announcement help-wanted
```text

This short syntax automatically expands to:
- Discussion labeled with any of the specified labels (question, announcement, help-wanted)
- Workflow dispatch trigger with an item_number input parameter

Note: GitHub Actions does not support the `names` field for discussion events, so label filtering
must be handled via job conditions if needed. The shorthand still provides workflow_dispatch support.

## Task

When this workflow is triggered, acknowledge that it was triggered and provide a brief summary.

Example output:
```text
✅ Workflow triggered!
📋 This workflow responds to label events on discussions
```
