---
name: PR Triage Agent
description: Labels pull requests based on change type when opened or updated
on:
  pull_request:
    types: [opened, reopened, edited, synchronize, ready_for_review]
permissions: read-all
tools:
  github:
    toolsets: [default]
safe-outputs:
  add-labels:
    max: 3
---

# PR Triage Agent

## Context

- **Repository**: ${{ github.repository }}
- **Pull Request**: #${{ github.event.pull_request.number }}
- **Title**: ${{ github.event.pull_request.title }}
- **Author**: @${{ github.actor }}

<!-- Edit the file linked below to modify the agent without recompilation. Feel free to move the entire markdown body to that file. -->
@./agentics/pr-triage-agent.md
