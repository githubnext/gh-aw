---
title: Using Claude Code with GitHub Agentic Workflows
description: Select and authenticate Claude Code as the AI engine for GitHub Agentic Workflows (gh-aw), understand its capabilities and limitations, and start from an example.
---

Claude Code is Anthropic's agentic coding interface for repository analysis and code changes. GitHub Agentic Workflows (`gh-aw`) runs Claude Code through GitHub Actions from a Markdown workflow, adds GitHub event triggers, and can route configured writes through validated safe outputs.

## Selection and authentication

Set `engine: claude` in workflow frontmatter and provide [`ANTHROPIC_API_KEY`](/gh-aw/reference/auth/#anthropic_api_key), or configure keyless [Anthropic Workload Identity Federation](/gh-aw/reference/auth/#anthropic-workload-identity-federation-wif). Claude subscription OAuth tokens such as `CLAUDE_CODE_OAUTH_TOKEN` are not supported.

## Example: scheduled repository report

```aw wrap title=".github/workflows/daily-status.md"
---
on:
  schedule: daily

permissions:
  contents: read
  issues: read
  pull-requests: read

engine: claude

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

Claude Code supports native web search, bare mode, top-level `max-turns`, and per-command bash allowlisting. It does not support Copilot-specific `max-continuations`, native `engine.agent` selection, or custom `engine.harness` scripts. See the [AI engine feature comparison](/gh-aw/reference/engines/#engine-feature-comparison).

## Guided workflow authoring with Claude Code

Claude Code users can initialize the repository and author agentic workflows with interactive guidance — no Copilot subscription required.

### Initialize for Claude

Run `gh aw init --engine claude` to configure the repository. The `--engine claude` flag skips Copilot-specific files (MCP server configuration, Copilot dispatcher skill) and only writes the files useful for any engine: `.gitattributes`, VS Code settings, and the custom agent file.

```bash
gh aw init --engine claude
```

### Create a workflow

Start Claude Code in your repository and run:

```text wrap
Create a workflow for GitHub Agentic Workflows using https://raw.githubusercontent.com/github/gh-aw/main/create.md

The purpose of the workflow is <describe your automation goal here>.
```

The agent fetches `create.md`, installs the `gh aw` CLI if needed, guides you through trigger selection, tools, safe outputs, and permissions, then generates `.github/workflows/<name>.md` and compiles it to a `.lock.yml`.

After the files are committed, set `engine: claude` in your workflow's frontmatter (if not already set) and add your `ANTHROPIC_API_KEY` as a repository secret.

> [!NOTE]
> The `agentic-workflows create` Copilot Chat skill (enabled by `gh aw init` without `--engine`) requires a GitHub Copilot subscription. For Claude Code-only setups, `gh aw init --engine claude` plus the `create.md` prompt above is the equivalent guided flow.

## GitHub Agentic Workflows vs. anthropics/claude-code-action

Choose GitHub Agentic Workflows when the workflow should be defined in Markdown, use the `gh-aw` security model, remain portable across built-in engines, or use safe outputs for validated GitHub writes. Choose [`anthropics/claude-code-action`](https://github.com/anthropics/claude-code-action) when the main goal is native Claude-driven PR assistance around comment and mention workflows.

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
