---
title: Using GitHub Copilot with GitHub Agentic Workflows
description: Select and authenticate GitHub Copilot as the AI engine for GitHub Agentic Workflows (gh-aw), understand its capabilities and limitations, and start from an example.
---

GitHub Agentic Workflows (`gh-aw`) uses GitHub Copilot as its default AI engine. GitHub Actions runs the Copilot agent from a Markdown workflow, while `gh-aw` supplies event routing, sandbox controls, and safe outputs for controlled repository writes.

## Selection and authentication

Set `engine: copilot` or omit `engine:` because Copilot is the default. Authenticate Copilot in one of these ways:

- For organization-billed usage, grant [`copilot-requests: write`](/gh-aw/reference/auth/#copilot-requests-write-permission).
- Otherwise, provide a [`COPILOT_GITHUB_TOKEN`](/gh-aw/reference/auth/#copilot_github_token) secret containing a fine-grained PAT with Copilot Requests access.

### Initialize the repository

Run `gh aw init` to configure the repository. Copilot is the default engine, so no `--engine` flag is required. This sets up the full Copilot integration: dispatcher skill, MCP server configuration, `.gitattributes`, VS Code settings, and the custom agent file.

```bash
gh aw init
```

## Example: scheduled repository report

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

## Capabilities and limitations

Copilot supports the broadest set of `gh-aw` engine-specific features: native custom-agent selection with `engine.agent`, custom harnesses, `max-continuations`, bare mode, and per-command bash allowlisting. Copilot CLI does not provide native `tools.web-search`; configure a supported MCP search integration when the workflow requires web search. See the [AI engine feature comparison](/gh-aw/reference/engines/#engine-feature-comparison).

## GitHub Agentic Workflows vs. native Copilot assignment

Choose GitHub Agentic Workflows for scheduled or event-driven automation, Markdown-defined workflows, and validated safe outputs. Choose GitHub Copilot's native `@copilot` assignment when the task is interactive PR-level coding assistance driven directly from the pull-request conversation.

## Related pages

- [Quick start](/gh-aw/setup/quick-start/)
- [Engine reference](/gh-aw/reference/engines/)
- [Authentication](/gh-aw/reference/auth/)
- [Security architecture](/gh-aw/introduction/architecture/)
- [Examples by task](/gh-aw/examples/)
- [AI issue triage](/gh-aw/guides/ai-issue-triage/)
- [Automated AI pull request review](/gh-aw/guides/automated-pr-review/)
- [AI-generated release notes and reports](/gh-aw/guides/ai-release-notes/)
- [Keeping documentation up to date automatically](/gh-aw/guides/docs-automation/)
