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

# Test Go File Size Reduction Campaign

Test the `update-project` safe output with the actual go-file-size-reduction campaign configuration.

## Campaign Configuration

- **Campaign ID**: `go-file-size-reduction`
- **Campaign Name**: Go File Size Reduction Campaign
- **Project URL**: https://github.com/orgs/githubnext/projects/60
- **Tracker Label**: `campaign:go-file-size-reduction`
- **Memory Paths**: `memory/campaigns/go-file-size-reduction-*/**`
- **Workflows**: daily-file-diet

## Test Tasks

Execute the following tests:

### Test 1: Verify Project Exists

Call `update-project` with just the project URL and campaign ID to verify the project exists and is accessible:

```
{
  "type": "update_project",
  "project": "https://github.com/orgs/githubnext/projects/60",
  "campaign_id": "go-file-size-reduction"
}
```

### Test 2: Add Issue to Campaign Project

Find the first open issue in the repository and add it to the campaign project:

```
{
  "type": "update_project",
  "project": "https://github.com/orgs/githubnext/projects/60",
  "campaign_id": "go-file-size-reduction",
  "content_type": "issue",
  "content_number": <issue_number>
}
```

### Test 3: Update Custom Fields

Update the issue on the project with custom campaign fields:

```
{
  "type": "update_project",
  "project": "https://github.com/orgs/githubnext/projects/60",
  "campaign_id": "go-file-size-reduction",
  "content_type": "issue",
  "content_number": <issue_number>,
  "fields": {
    "Status": "In Progress",
    "Priority": "High"
  }
}
```

### Test 4: Verify Campaign Label

Use the GitHub API to verify the issue has the campaign label `campaign:go-file-size-reduction`.

### Test 5: Orchestrator Pattern

Test the pattern used by the campaign orchestrator (just project URL and campaign ID with no issue):

```
{
  "type": "update_project",
  "project": "https://github.com/orgs/githubnext/projects/60",
  "campaign_id": "go-file-size-reduction"
}
```

## Expected Results

- All tests should pass without errors
- The issue should appear on the campaign project board
- The issue should have the `campaign:go-file-size-reduction` label
- Custom fields should be set (if they exist in the project)

## Notes

- If custom fields don't exist, they need to be created in the GitHub UI first
- The project must already exist at the specified URL
- Requires `projects: write` permission
