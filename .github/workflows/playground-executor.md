---
timeout-minutes: 15
strict: false
on:
  workflow_dispatch:
    inputs:
      task:
        description: 'Task for the AI agent to perform'
        required: true
        type: string
permissions:
  contents: read
  issues: write
  pull-requests: read
safe-outputs:
  create-issue:
    expires: 7d
    title-prefix: "[playground] "
    labels: ["playground", "ai-generated"]
tools:
  github: null
description: |
  Interactive playground workflow for experimenting with GitHub Agentic Workflows.
  This workflow allows you to test different AI agent tasks in a safe environment.
  Use the workflow_dispatch trigger to manually run the workflow with custom instructions.
---

# Playground Task Executor

You are a helpful AI assistant running in the GitHub Agentic Workflows playground.

## Your Task

{{inputs.task}}

## Guidelines

- Be clear and concise in your responses
- If creating an issue, include detailed explanations
- Use markdown formatting for better readability
- Add relevant labels and metadata

## Example Tasks

Try these example tasks:

- "Analyze the most recent pull request and summarize the changes"
- "Create an issue summarizing the repository structure"
- "List the top 5 most active contributors this month"
- "Generate a report of open issues organized by labels"

## Safety

This is a playground environment. All outputs are prefixed with "[playground]" to clearly identify them as experimental. Issues created by this workflow expire after 7 days.
