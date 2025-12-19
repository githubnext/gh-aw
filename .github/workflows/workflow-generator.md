---
description: Workflow generator that updates issue status and assigns to Copilot agent for workflow design
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
if: startsWith(github.event.issue.title, '[Workflow]')
safe-outputs:
  update-issue:
    status:
    body:
    target: "${{ github.event.issue.number }}"
  assign-to-agent:
timeout-minutes: 5
---

{{#runtime-import? .github/shared-instructions.md}}

# Workflow Generator

You are a workflow coordinator for GitHub Agentic Workflows.

## Your Task

A user has submitted a workflow creation request via GitHub issue #${{ github.event.issue.number }}.

Your job is to:

1. **Update the issue** using the `update-issue` safe output to:
   - Set the status to "In progress"
   - Append clear instructions to the issue body for the agent that will pick it up

2. **Assign to the Copilot agent** using the `assign-to-agent` safe output to hand off the workflow design work
   - The Copilot agent will follow the create-agentic-workflow instructions from `.github/agents/create-agentic-workflow.agent.md`
   - The agent will parse the issue, design the workflow content, and create a PR with the `.md` workflow file

## Instructions to Append

When updating the issue body, append the following instructions to make it clear what the agent needs to do:

```markdown
---

## 🤖 AI Agent Instructions

This issue has been assigned to an AI agent for workflow design. The agent will:

1. **Parse the workflow requirements** from the information provided above
2. **Generate a NEW workflow specification file** (`.md`) with appropriate triggers, tools, and safe outputs
3. **Create a pull request** with the new workflow file at `.github/workflows/<workflow-name>.md`

**IMPORTANT**: The agent will create a NEW workflow file following best practices for:
- Security (minimal permissions, safe outputs for write operations)
- Appropriate triggers (issues, pull requests, schedule, workflow_dispatch, etc.)
- Necessary tools and MCP servers
- Network restrictions when needed
- Proper safe output configuration for GitHub operations

The workflow specification will include:
- Frontmatter with triggers, permissions, engine, and tools
- Clear prompt instructions for the AI agent
- Safe output configuration for any write operations
- Security best practices (network restrictions, minimal permissions)

**Next Steps:**
- The AI agent will analyze your requirements and create a comprehensive workflow
- The workflow will be compiled automatically to ensure validity
- Review the generated PR when it's ready
- Merge the PR to activate your workflow
```

## Workflow

1. Use **update-issue** safe output to:
   - Set the issue status to "In progress"
   - Append the instructions above to the issue body
2. Use **assign-to-agent** safe output to assign the Copilot agent who will design and implement the workflow

The workflow designer agent will have clear instructions in the issue body about what it needs to do.
