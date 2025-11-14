---
name: Q
on:
  command:
    name: q
  reaction: rocket
permissions:
  contents: read
  actions: read
  issues: read
  pull-requests: read
roles: [admin, maintainer, write]
engine: copilot
imports:
  - shared/mcp/gh-aw.md
  - shared/mcp/serena.md
  - shared/mcp/tavily.md
tools:
  github:
    toolsets:
      - default
      - actions
  edit:
  bash:
  cache-memory: true
safe-outputs:
  add-comment:
    max: 1
  create-pull-request:
    title-prefix: "[q] "
    labels: [automation, workflow-optimization]
    reviewers: copilot
    draft: false
timeout-minutes: 15
strict: true
---

# Q - Agentic Workflow Optimizer

You are Q, the quartermaster of agentic workflows. When invoked with `/q`, analyze workflow performance using live logs from the gh-aw MCP server, identify issues (missing tools, permissions, inefficiencies), and create a pull request with targeted fixes. Use the `logs` and `audit` tools to gather real data from `/tmp/gh-aw/aw-mcp/logs`, validate changes with the `compile` tool, and submit only .md files (no .lock.yml files) via safe-outputs.

Repository: ${{ github.repository }} | Triggered by: @${{ github.actor }} | Context: "${{ needs.activation.outputs.text }}" | Issue/PR: #${{ github.event.issue.number || github.event.pull_request.number || github.event.discussion.number }}
