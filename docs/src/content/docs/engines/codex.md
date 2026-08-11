---
title: Using OpenAI Codex with GitHub Agentic Workflows
description: Select and authenticate OpenAI Codex as the AI engine for GitHub Agentic Workflows (gh-aw), understand its capabilities and limitations, and start from an example.
---

OpenAI Codex is OpenAI's coding-focused agent runtime for repository work. GitHub Agentic Workflows (`gh-aw`) runs Codex through GitHub Actions from a Markdown workflow and adds GitHub triggers, sandbox controls, and safe outputs for event-driven, reviewable automation.

## Selection and authentication

Set `engine: codex` and provide `CODEX_API_KEY` or [`OPENAI_API_KEY`](/gh-aw/reference/auth/#openai_api_key). `CODEX_API_KEY` takes precedence when both secrets are present.

### Initialize the repository

Run `gh aw init --engine codex` to configure the repository. The `--engine codex` flag skips Copilot-specific files (MCP server configuration, Copilot dispatcher skill) and writes only the files useful for any engine: `.gitattributes`, VS Code settings, and the custom agent file.

```bash
gh aw init --engine codex
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

## Capabilities and limitations

Codex supports native web search when `tools.web-search` is enabled and can disable shell execution completely. Codex cannot enforce a nonempty per-command `tools.bash` allowlist and does not support bare mode, `max-continuations`, native `engine.agent` selection, or custom `engine.harness` scripts. See the [AI engine feature comparison](/gh-aw/reference/engines/#engine-feature-comparison).

## GitHub Agentic Workflows vs. running Codex directly in Actions

Choose GitHub Agentic Workflows when the workflow should stay in Markdown, reuse the same structure across engines, and route configured GitHub writes through safe outputs. Run Codex directly in Actions when the job needs a custom script layout and manual control over every integration point.

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
