---
engine: copilot
on:
  workflow_dispatch:
permissions:
  contents: read
  issues: write
---

# Test Project Board Safe Outputs

Test the new project board safe output types.

## Task

Create a simple test to verify project board safe outputs work:

1. Output a `create-project` safe output to create a project called "Test Project Board"
2. Output an `add-project-item` safe output to add a draft item
3. Output an `update-project-item` safe output to update the item status

Use this exact format for safe outputs:

```json
{
  "type": "create-project",
  "title": "Test Project Board",
  "description": "Testing project board safe outputs"
}
```

```json
{
  "type": "add-project-item",
  "project": "Test Project Board",
  "content_type": "draft",
  "title": "Test Draft Item",
  "body": "This is a test draft item",
  "fields": {
    "Status": "To Do"
  }
}
```

**Note**: These outputs will be validated against the schema but handlers are not yet implemented.
