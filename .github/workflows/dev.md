---
on: 
  workflow_dispatch:
name: Dev
description: Test go-file-size-reduction campaign update-project intrinsic
timeout-minutes: 10
strict: false
engine: copilot

permissions:
  contents: read
  issues: write

safe-outputs:
  update-project:
    max: 10

tools:
  github:
    toolsets: [default]
---

# Test update-project Safe Output

**CRITICAL**: The `update-project` safe output requires SPECIFIC FIELDS. You MUST provide them.

## What NOT to do

❌ **WRONG** - This will FAIL validation:
```json
{"type": "update_project"}
```

## What TO do

✅ **CORRECT** - Provide ALL required fields:
```json
{
  "type": "update_project",
  "project": "https://github.com/orgs/githubnext/projects/60",
  "campaign_id": "go-file-size-reduction",
  "content_type": "issue",
  "content_number": 999
}
```

## Your Task

1. **List open issues** in this repository using the GitHub tool
2. **Find the first open issue number** (e.g., 123, 456, etc.)
3. **Output the COMPLETE JSON** shown above, replacing `999` with the actual issue number

## Required Fields

- `type`: MUST be "update_project"
- `project`: MUST be "https://github.com/orgs/githubnext/projects/60" (exact URL)
- `campaign_id`: MUST be "go-file-size-reduction"  
- `content_type`: MUST be "issue"
- `content_number`: MUST be an actual issue number from this repo

## Notes

- The `project` field is REQUIRED and MUST be the full GitHub project URL
- If you output just `{"type":"update_project"}`, validation will fail
- Get a real issue number from the GitHub API first
