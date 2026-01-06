---
name: Copy Project Template
description: Example workflow demonstrating copy-project with configured defaults
on:
  workflow_dispatch:
    inputs:
      project_title:
        description: Title for the new project
        required: true
        type: string

engine: copilot

safe-outputs:
  copy-project:
    source-project: "https://github.com/orgs/myorg/projects/1"
    target-owner: "myorg"
    github-token: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}
    max: 1
---

# Copy Project Template

This workflow demonstrates how to use the `copy-project` safe output with configured defaults.

## How it works

With the `source-project` and `target-owner` configured in the frontmatter, the agent only needs to provide the `title` field when calling the `copy_project` tool. The workflow will automatically use the configured defaults for the source project and target owner.

## Example output

```javascript
// Agent can simply provide the title
copy_project({
  title: "${{ github.event.inputs.project_title }}"
});

// Or override the defaults if needed
copy_project({
  sourceProject: "https://github.com/orgs/otherorg/projects/5",
  owner: "differentorg",
  title: "${{ github.event.inputs.project_title }}"
});
```

## Benefits

- **Simplified agent calls**: Agent doesn't need to know the source project URL
- **Centralized configuration**: Project template source is defined once in the workflow
- **Flexibility**: Agent can still override defaults when needed
- **Reusability**: Easy to create multiple projects from the same template

## Use cases

- Creating sprint projects from a template
- Setting up new team boards
- Duplicating project structures across organizations
- Automated project provisioning
