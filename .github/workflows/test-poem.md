---
on:
  workflow_dispatch:
  issues:
    types: [opened, edited]
  pull_request:
    types: [opened, edited, synchronize]

safe-outputs:
  missing-tool:
    max: 5
  staged: true

tools:
  cache-memory: true

engine: claude

permissions: read-all
---

# Test Poem Workflow

This workflow generates a poem about the current changes or issues in the repository and demonstrates the `missing-tool` safe output functionality using Claude AI.

## Purpose

This workflow validates the missing-tool safe output type by:
- Triggering on workflow dispatch, issue events, and pull request events
- Using Claude AI to generate contextual poems about repository state
- Demonstrating Claude's creative capabilities with repository context
- Using staged mode to prevent actual GitHub interactions
- Demonstrating memory persistence with cache-memory integration

## Trigger Events

- **workflow_dispatch**: Manual execution for testing
- **issues.opened/edited**: Responds to issue creation and updates
- **pull_request.opened/edited/synchronize**: Responds to PR events

## Safe Output Configuration

- **staged: true**: Prevents real GitHub interactions
- **max: 5**: Allows up to 5 missing tool reports per workflow run

## Tools

- **cache-memory**: Enables persistent memory across workflow runs for improved context and learning

## Claude Engine Implementation

The workflow uses Claude AI to:
1. Analyze the current repository context (issues, PRs, or general state)
2. Generate creative, contextual poetry about the changes
3. Utilize memory from previous runs to improve continuity
4. Provide sophisticated natural language understanding of code changes
5. Create meaningful verse that reflects the actual repository activity

This demonstrates how Claude can be used for creative applications while leveraging memory for enhanced context awareness across multiple workflow executions.

## Example Use Cases

- **Issue Opened**: Creates a poem analyzing the new issue and its implications
- **Pull Request**: Generates verse about the proposed changes and their impact
- **Manual Dispatch**: Creates contextual poetry about the current repository state

The workflow showcases Claude's ability to understand technical context and transform it into creative output while maintaining memory across executions for improved continuity.