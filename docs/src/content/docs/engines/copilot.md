---
title: Run GitHub Copilot agents in GitHub Actions with gh-aw
description: Configure gh-aw to run GitHub Copilot agents in GitHub Actions using Markdown workflows, GitHub triggers, and safe outputs.
---

GitHub Copilot is the default gh-aw engine and runs GitHub Copilot agents inside a GitHub Actions workflow. gh-aw provides the Markdown workflow format, event routing, sandbox controls, and safe outputs so the agent can analyze repository state and propose changes without direct write access.

## Setup

Set `engine: copilot` or omit `engine:` because Copilot is the default. For organization-billed usage, grant `copilot-requests: write`; otherwise provide a `COPILOT_GITHUB_TOKEN` secret. See [Copilot authentication](/gh-aw/reference/auth/#copilot_github_token) for both paths.

### Initialize the repository

Run `gh aw init` to configure the repository. Copilot is the default engine, so no `--engine` flag is required. This sets up the full Copilot integration: dispatcher skill, MCP server configuration, `.gitattributes`, VS Code settings, and the custom agent file.

```bash
gh aw init
```

```aw wrap title=".github/workflows/daily-status.md"
---
on:
  schedule: daily

permissions:
  contents: read
  issues: read
  pull-requests: read
  copilot-requests: write

engine: copilot

safe-outputs:
  create-issue:
    title-prefix: "[status] "
    labels: [report]
    close-older-issues: true
---

# Daily Repository Status

Analyze the repository and create a concise daily status report covering:
- Open issues and their priority
- Recent PR activity
- Upcoming work items
```

## When to choose gh-aw vs. GitHub Copilot's native @copilot assignment

Choose gh-aw for scheduled or event-driven automation, Markdown-defined workflows, and validated safe outputs. Choose GitHub Copilot's native `@copilot` assignment when the task is interactive PR-level coding assistance driven directly from the pull request conversation.

## Related pages

- [Quick start](/gh-aw/setup/quick-start/)
- [Engine reference](/gh-aw/reference/engines/)
- [AI issue triage](/gh-aw/guides/ai-issue-triage/)
- [Automated AI pull request review](/gh-aw/guides/automated-pr-review/)
- [AI-generated release notes and reports](/gh-aw/guides/ai-release-notes/)
- [Keeping documentation up to date automatically](/gh-aw/guides/docs-automation/)
