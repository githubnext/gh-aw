---
name: Security Alert Burndown
description: Discovers Dependabot PRs and assigns them to Copilot for review
on:
  schedule:
    - cron: "0 * * * *"
  workflow_dispatch:
permissions:
  issues: read
  pull-requests: read
  contents: read
project:
  url: https://github.com/orgs/githubnext/projects/134
---

# Security Alert Burndown Campaign

This campaign discovers Dependabot-created pull requests for JavaScript dependencies and assigns them to the Copilot coding agent for automated review and merging.

## Objective

Systematically process Dependabot dependency update PRs to keep JavaScript dependencies up-to-date and secure.

## Discovery Strategy

Discover Dependabot pull requests with labels: `dependencies`, `javascript`.

Prioritize open PRs by age (oldest first). Skip items already marked "Done" on the project board.

## Campaign Execution

Each run discovers and processes Dependabot PRs, updating the project board with current status.

## Success Criteria

- JavaScript dependency PRs are automatically triaged
- Copilot agent reviews and processes assigned PRs
- Project board reflects current state of dependency updates
