---
title: Using Google Gemini with GitHub Agentic Workflows
description: Select and authenticate Google Gemini as the AI engine for GitHub Agentic Workflows (gh-aw), understand its capabilities and limitations, and start from an example.
---

Google Gemini is Google's model family for coding and repository analysis. GitHub Agentic Workflows (`gh-aw`) runs the Gemini CLI through GitHub Actions from a Markdown workflow and adds GitHub event triggers, sandbox controls, and safe outputs for constrained, reviewable automation.

## Selection and authentication

Set `engine: gemini` and provide [`GEMINI_API_KEY`](/gh-aw/reference/auth/#gemini_api_key), or configure keyless [Google Workload Identity Federation](/gh-aw/reference/auth/#google-workload-identity-federation-wif).

### Initialize the repository

Run `gh aw init --engine gemini` to configure the repository. The `--engine gemini` flag skips Copilot-specific files (MCP server configuration, Copilot dispatcher skill) and writes only the files useful for any engine: `.gitattributes`, VS Code settings, and the custom agent file.

```bash
gh aw init --engine gemini
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

engine: gemini

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

Gemini supports top-level `max-turns`, custom API targets, and per-command bash allowlisting. Gemini does not provide native `tools.web-search`; configure an MCP search integration when needed. It also does not support bare mode, `max-continuations`, native `engine.agent` selection, or custom `engine.harness` scripts. See the [AI engine feature comparison](/gh-aw/reference/engines/#engine-feature-comparison).

## GitHub Agentic Workflows vs. running Gemini directly in Actions

Choose GitHub Agentic Workflows when the workflow should be authored in Markdown, share one structure across engines, and route configured GitHub writes through validated safe outputs. Run Gemini directly in Actions when the job needs a custom script pipeline and all security and review controls are managed manually.

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
