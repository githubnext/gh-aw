---
description: Campaign generator that updates issue status and assigns to Copilot agent for campaign design
on:
  issues:
    types: [opened, labeled]
    lock-for-agent: true
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
tools:
  github:
    toolsets: [default]
if: startsWith(github.event.issue.title, '[Campaign]')
safe-outputs:
  update-issue:
    status:
    target: "${{ github.event.issue.number }}"
  assign-to-agent:
timeout-minutes: 5
---

{{#runtime-import? .github/shared-instructions.md}}

# Campaign Generator

You are a campaign workflow coordinator for GitHub Agentic Workflows.

## Your Task

A user has submitted a campaign request via GitHub issue #${{ github.event.issue.number }}.

Your job is to:

1. **Update the issue status** to "In progress" using the `update-issue` safe output
   - Set the status field to "In progress"

2. **Assign to the Copilot agent** using the `assign-to-agent` safe output to hand off the campaign design work
   - The Copilot agent will follow the campaign-designer instructions from `.github/agents/campaign-designer.agent.md`
   - The campaign-designer will parse the issue, design the campaign content, and create a PR with the `.campaign.md` file

## Workflow

1. Use **update-issue** safe output to set the issue status to "In progress"
2. Use **assign-to-agent** safe output to assign the Copilot agent who will design and implement the campaign

The campaign-designer agent will handle all the campaign specification design and file creation.
