---
description: Test update-project with content_repo for cross-repo project item resolution
on:
  workflow_dispatch:
    inputs:
      project_url:
        description: "GitHub Projects v2 URL (e.g., https://github.com/orgs/myorg/projects/42)"
        required: true
        type: string
      source_repo:
        description: "Source repository for issues (e.g., myorg/other-repo)"
        required: true
        type: string
      issue_number:
        description: "Issue number in the source repository"
        required: true
        type: string
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
tools:
  github:
    mode: remote
    toolsets: [default]
safe-outputs:
  update-project:
    github-token: ${{ secrets.GH_AW_PROJECT_GITHUB_TOKEN }}
    project: "https://github.com/orgs/<ORG>/projects/<NUMBER>"
    allowed-repos:
      - ${{ inputs.source_repo }}
---

# Test Update Project with Cross-Repo Content Resolution

This workflow tests the `content_repo` field on the `update_project` safe output, which
enables updating project fields for issues and pull requests that originate from repositories
other than the workflow's host repository.

This is useful for organization-level projects that aggregate issues from multiple repos.

## Task

Use the `update_project` tool to add issue #${{ inputs.issue_number }} from `${{ inputs.source_repo }}` to the project at `${{ inputs.project_url }}` and set its status to "In Progress".

You must include `content_repo: "${{ inputs.source_repo }}"` in the `update_project` output to resolve the issue from the correct repository.

Example output:
```json
{
  "type": "update_project",
  "project": "${{ inputs.project_url }}",
  "content_type": "issue",
  "content_number": ${{ inputs.issue_number }},
  "content_repo": "${{ inputs.source_repo }}",
  "fields": {
    "Status": "In Progress"
  }
}
```

After updating the project item, report what you did.
