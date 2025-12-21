---
name: Label Trigger Example - Comma Separated
description: Example workflow demonstrating the comma-separated label trigger syntax
on: issue labeled bug, enhancement, priority-high
engine:
  id: codex
  model: gpt-5-mini
strict: true
---

# Label Trigger Example - Comma Separated

This workflow demonstrates the comma-separated label trigger shorthand syntax:

```yaml
on: issue labeled bug, enhancement, priority-high
```

This short syntax automatically expands to:
- Issues labeled with any of the specified labels (bug, enhancement, or priority-high)
- Workflow dispatch trigger with an item_number input parameter

## Task

When this workflow is triggered, acknowledge that it was triggered and provide a brief summary.

Example output:
```
✅ Workflow triggered!
📋 This workflow responds to label events on issues
🏷️ Triggered by one of: bug, enhancement, priority-high
```
