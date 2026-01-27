---
name: Security Alert Burndown
description: Discovers Dependabot PRs and assigns them to Copilot for review
on:
  schedule:
    - cron: "0 * * * *"
  workflow_dispatch:
project:
  url: https://github.com/orgs/githubnext/projects/134
  scope:
    - githubnext/gh-aw
  id: security-alert-burndown
  governance:
    max-new-items-per-run: 3
    max-discovery-items-per-run: 100
    max-discovery-pages-per-run: 5
    max-project-updates-per-run: 10
---

# Security Alert Burndown Campaign

This campaign discovers Dependabot-created pull requests for JavaScript dependencies and assigns them to the Copilot coding agent for automated review and merging.

## Objective

Systematically process Dependabot dependency update PRs to keep JavaScript dependencies up-to-date and secure.

## Discovery Strategy

The orchestrator will:

1. **Discover** pull requests opened by the `dependabot` bot
2. **Filter** to PRs with labels `dependencies` and `javascript`
3. **Assign** discovered PRs to the Copilot coding agent using `assign-to-agent`
4. **Track** progress in the project board

## Campaign Execution

Each run:
- Discovers up to 100 Dependabot PRs with specified labels
- Processes up to 5 pages of results
- Assigns up to 3 new items to Copilot
- Updates project board with up to 10 status changes

## Success Criteria

- JavaScript dependency PRs are automatically triaged
- Copilot agent reviews and processes assigned PRs
- Project board reflects current state of dependency updates
