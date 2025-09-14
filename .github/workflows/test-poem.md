---
on:
  push:
    branches:
      - 'copilot/*'

safe-outputs:
  missing-tool:
    max: 5
  staged: true

tools:
  cache-memory: true

engine:
  id: claude
  max-turns: 5

permissions: read-all
---

# Test Poem Workflow

This workflow generates a poem about the current changes or issues in the repository and demonstrates the `missing-tool` safe output functionality using Claude AI with a maximum of 5 turns.

## Purpose

This workflow validates the missing-tool safe output type by:
- Triggering on pushes to copilot/* branches
- Using Claude AI to generate contextual poems about repository state
- Demonstrating Claude's creative capabilities with repository context
- Using staged mode to prevent actual GitHub interactions
- Demonstrating memory persistence with cache-memory integration
- Limiting conversation to 5 turns for focused interaction

## Trigger Events

- **push.copilot/***: Triggers on pushes to any branch starting with "copilot/"

## Safe Output Configuration

- **staged: true**: Prevents real GitHub interactions
- **max: 5**: Allows up to 5 missing tool reports per workflow run

## Engine Configuration

- **engine: claude**: Uses Claude AI for natural language processing
- **max-turns: 5**: Limits the conversation to 5 turns for focused interaction

## Tools

- **cache-memory**: Enables persistent memory across workflow runs for improved context and learning

## Claude Engine Implementation

The workflow uses Claude AI to:
1. Analyze the current repository context from copilot branch changes
2. Generate creative, contextual poetry about the changes
3. Utilize memory from previous runs to improve continuity
4. Provide sophisticated natural language understanding of code changes
5. Create meaningful verse that reflects the actual repository activity
6. Complete the task within 5 conversation turns for efficiency

This demonstrates how Claude can be used for creative applications while leveraging memory for enhanced context awareness across multiple workflow executions with controlled conversation length.

## Example Use Cases

- **Copilot Branch Push**: Creates a poem analyzing the new changes pushed to copilot branches
- **Feature Development**: Generates verse about ongoing development work in copilot branches
- **Code Review Preparation**: Creates contextual poetry about changes ready for review

The workflow showcases Claude's ability to understand technical context and transform it into creative output while maintaining memory across executions for improved continuity, all within a focused 5-turn conversation.