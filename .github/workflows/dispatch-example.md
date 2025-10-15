---
on:
  issues:
    types: [opened]
permissions:
  contents: read
  actions: read
  issues: read
engine: copilot
safe-outputs:
  dispatch-workflow:
    allowed-workflows:
      - ci.yml
      - deploy-staging.yml
      - deploy-production.yml
    max: 1
timeout_minutes: 10
---

# Issue Workflow Dispatcher

This workflow analyzes newly opened issues and automatically dispatches the appropriate workflow based on the issue content.

## Task

1. Read the issue title and body from ${{ needs.activation.outputs.text }}
2. Analyze the content to determine which workflow should be triggered:
   - If the issue mentions "bug" or "fix", dispatch **ci.yml**
   - If the issue mentions "deploy" and "staging", dispatch **deploy-staging.yml**
   - If the issue mentions "deploy" and "production", dispatch **deploy-production.yml**
3. Use the `dispatch-workflow` tool from safe-outputs MCP to trigger the appropriate workflow

## Available Workflows

You can only dispatch workflows from this allowed list:
- ci.yml - Runs CI tests
- deploy-staging.yml - Deploys to staging environment
- deploy-production.yml - Deploys to production environment

## Notes

- The dispatch-workflow tool validates that the workflow name is in the allowed list
- You can pass inputs to the workflow if needed
- The workflow must have `workflow_dispatch` trigger to be dispatchable
