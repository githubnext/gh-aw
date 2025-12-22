---
name: Label Trigger Example - Comma No Spaces
description: Example workflow demonstrating comma-separated labels without spaces
on: issue labeled bug,enhancement,priority-high
engine:
  id: codex
  model: gpt-5-mini
strict: true
---

# Label Trigger Example - Comma No Spaces

This workflow tests the syntax without spaces after commas:

```yaml
on: issue labeled bug,enhancement,priority-high
```

## Task

Acknowledge the trigger.
