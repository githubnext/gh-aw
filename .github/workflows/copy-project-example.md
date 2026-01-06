---
name: Copy Project Example
engine: copilot
safe-outputs:
  copy-project:
    github-token: ${{ secrets.PROJECTS_PAT }}
---

# Copy Project Example

This workflow demonstrates how to copy a GitHub Projects V2 board using the new `copy_project` safe output.

## What it does

The workflow will:
1. Take a source project URL
2. Copy it to a new project under the specified owner
3. Return the new project's ID, title, and URL

## Example Usage

Use the `copy_project` tool to copy a project:

```javascript
copy_project({
  sourceProject: "https://github.com/orgs/myorg/projects/42",
  owner: "myorg",
  title: "Copied Project Template"
})
```

## Parameters

- **sourceProject**: Full GitHub project URL (e.g., `https://github.com/orgs/myorg/projects/42`)
- **owner**: Login name of the organization or user that will own the new project
- **title**: Title for the new project

## Notes

- This requires a GitHub token with Projects access (classic PAT with `project` scope or fine-grained PAT with Projects permission)
- Draft issues from the source project are NOT copied
- Custom fields, views, and workflows ARE copied
