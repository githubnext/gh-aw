---
title: Run Google Gemini in GitHub Actions with gh-aw
description: Configure gh-aw to run Google Gemini in GitHub Actions with Markdown workflows, GitHub triggers, and safe outputs.
---

Google Gemini is Google's model family for coding and repository analysis. gh-aw runs Gemini inside GitHub Actions from a Markdown workflow and adds GitHub event triggers, sandboxing controls, and safe outputs so automated repository work can stay constrained and reviewable.

## Setup

Set `engine: gemini` and provide `GEMINI_API_KEY`, or configure keyless authentication with Google Workload Identity Federation. See [Gemini authentication](/gh-aw/reference/auth/#gemini_api_key) for the available setup paths.

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

## When to choose gh-aw vs. running Gemini directly in Actions

Choose gh-aw when the workflow should be authored in Markdown, share one structure across engines, and restrict GitHub writes to validated safe outputs. Run Gemini directly in Actions when the job needs a custom script pipeline and all security and review controls are managed manually.

## Related pages

- [Quick start](/gh-aw/setup/quick-start/)
- [Engine reference](/gh-aw/reference/engines/)
- [AI issue triage](/gh-aw/guides/ai-issue-triage/)
- [Automated AI pull request review](/gh-aw/guides/automated-pr-review/)
- [AI-generated release notes and reports](/gh-aw/guides/ai-release-notes/)
- [Keeping documentation up to date automatically](/gh-aw/guides/docs-automation/)
