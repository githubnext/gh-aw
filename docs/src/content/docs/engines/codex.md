---
title: Run OpenAI Codex in GitHub Actions with gh-aw
description: Configure gh-aw to run OpenAI Codex in GitHub Actions with Markdown workflows, GitHub triggers, and safe outputs.
---

OpenAI Codex is OpenAI's coding-focused agent runtime for repository work. gh-aw runs Codex inside GitHub Actions from a Markdown workflow and layers GitHub triggers, sandbox controls, and safe outputs on top so repository automation can stay event-driven and reviewable.

## Setup

Set `engine: codex` and provide `OPENAI_API_KEY`. `CODEX_API_KEY` is also accepted and takes precedence when both are present. See [OpenAI authentication](/gh-aw/reference/auth/#openai_api_key) for the supported configuration.

### Initialize the repository

Run `gh aw init --engine codex` to configure the repository. The `--engine codex` flag skips Copilot-specific files (MCP server configuration, Copilot dispatcher skill) and writes only the files useful for any engine: `.gitattributes`, VS Code settings, and the custom agent file.

```bash
gh aw init --engine codex
```

```aw wrap title=".github/workflows/daily-status.md"
---
on:
  schedule: daily

permissions:
  contents: read
  issues: read
  pull-requests: read

engine: codex

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

## When to choose gh-aw vs. running Codex directly in Actions

Choose gh-aw when the workflow should stay in Markdown, reuse the same structure across engines, and enforce GitHub writes through safe outputs. Run Codex directly in Actions when the job needs a custom script layout and manual control over every integration point.

## Related pages

- [Quick start](/gh-aw/setup/quick-start/)
- [Engine reference](/gh-aw/reference/engines/)
- [AI issue triage](/gh-aw/guides/ai-issue-triage/)
- [Automated AI pull request review](/gh-aw/guides/automated-pr-review/)
- [AI-generated release notes and reports](/gh-aw/guides/ai-release-notes/)
- [Keeping documentation up to date automatically](/gh-aw/guides/docs-automation/)
