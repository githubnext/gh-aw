---
title: Run Claude Code in GitHub Actions with gh-aw
description: Configure gh-aw to run Claude Code in GitHub Actions with Markdown workflows, safe outputs, and standard GitHub triggers.
---

Claude Code is Anthropic's agentic coding interface for repository analysis and code changes. gh-aw runs Claude Code inside GitHub Actions from a Markdown workflow, adds GitHub event triggers, and routes write operations through validated safe outputs instead of giving the agent direct write access.

## Setup

Set `engine: claude` in the workflow frontmatter and provide `ANTHROPIC_API_KEY`, or configure keyless authentication with Anthropic Workload Identity Federation. See [Anthropic authentication](/gh-aw/reference/auth/#anthropic_api_key) for the supported options.

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

## When to choose gh-aw vs. anthropics/claude-code-action

Choose gh-aw when the workflow should be defined in Markdown, run behind gh-aw sandboxing, switch engines without rewriting the workflow, or use safe outputs for validated GitHub writes. Choose [`anthropics/claude-code-action`](https://github.com/anthropics/claude-code-action) when the main goal is native Claude-driven PR assistance around comment and mention workflows.

## Related pages

- [Quick start](/gh-aw/setup/quick-start/)
- [Engine reference](/gh-aw/reference/engines/)
- [AI issue triage](/gh-aw/guides/ai-issue-triage/)
- [Automated AI pull request review](/gh-aw/guides/automated-pr-review/)
- [AI-generated release notes and reports](/gh-aw/guides/ai-release-notes/)
- [Keeping documentation up to date automatically](/gh-aw/guides/docs-automation/)
