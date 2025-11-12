---
name: Bug Bash Campaign
on:
  schedule:
    - cron: "0 10 * * 1"  # Every Monday at 10am - kick off the weekly bug bash
  workflow_dispatch:
    inputs:
      project_url:
        description: "GitHub project URL (org or user). Examples: https://github.com/orgs/ACME/projects/42 | https://github.com/users/alice/projects/19"
        required: true
        type: string

engine: copilot

permissions:
  contents: read
  issues: write
  repository-projects: write
  pull-requests: read

safe-outputs:
  update-project:
    github-token: ${{ secrets.PROJECT_GITHUB_TOKEN }}
    max: 15

tools:
  github:
    mode: remote
    toolsets: [default]

timeout-minutes: 10
---

# Bug Bash Campaign - Weekly Sprint

You are the Bug Bash Campaign orchestrator. Every week, you organize a focused bug hunting session.

## Steps

1. Ensure the board exists (create if missing) using `project_url`.
2. Find recent open issues (last 30 days) with labels: `bug`, `defect`, or `regression` that are not already on the board and not closed.
3. For each selected issue emit an `update-project` safe output with fields:
   - Status: "To Do"
   - Priority: "Critical" if P0/P1 label, else "High" if multiple comments/reactions (>=3), else "Medium".
   - Complexity: "Quick Win" if short/simple (<600 chars body) else "Standard" otherwise; use "Complex" only if label `architecture` or `security` present.
   - Impact: "Blocker" if blocking major feature (label `blocker`), else "Major" if multiple area/component labels, else "Minor".
4. Limit additions to `max` (15) in safe-outputs.
5. Create one summary issue with:
   - Count scanned vs added
   - Top 3 critical items (number + title)
   - Any quick wins (list numbers)

## Guardrails
- Skip items with `enhancement` label unless they also have a bug label.
- Do not modify items in progress.
- Use `${{ needs.activation.outputs.text }}` for any manual context (if dispatched from an issue).

## Example
```json
{
  "type": "update-project",
  "project": "Bug Bash 2025",
  "content_type": "issue",
  "content_number": 123,
  "fields": {
    "Status": "To Do",
    "Priority": "High",
    "Complexity": "Standard",
    "Impact": "Major"
  }
}
```
