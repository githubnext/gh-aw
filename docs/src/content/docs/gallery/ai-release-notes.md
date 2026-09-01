---
title: AI-generated release notes and reports
description: Use gh-aw to generate release notes and release reports automatically from GitHub events with validated safe outputs.
---

AI-generated release notes with GitHub Agentic Workflows means running an agent when a release is published, a branch is pushed, or a scheduled reporting window arrives so it can summarize merged work and produce a publishable report. gh-aw handles the GitHub-side output through safe outputs instead of direct write access from the agent.

Install the starter with `gh aw add-wizard githubnext/agentics/release-notes`.

```aw wrap title=".github/workflows/release-notes.md"
---
on:
  release:
    types: [published]

permissions:
  contents: read
  pull-requests: read
  issues: read

safe-outputs:
  create-issue:
    title-prefix: "[release-notes] "
    labels: [release-notes]
    close-older-issues: true
---

# Release Notes Generator

Generate release notes for the published release.

Summarize the most important merged pull requests, user-visible fixes, documentation updates, and any upgrade or rollback notes. Create one issue containing the draft release notes and a short follow-up checklist for human review before publication elsewhere.
```

## Learn More

- [Run Claude Code in GitHub Actions with gh-aw](/gh-aw/engines/claude/)
- [Run GitHub Copilot agents in GitHub Actions with gh-aw](/gh-aw/engines/copilot/)
- [Engine reference](/gh-aw/reference/engines/)
- [Quick start](/gh-aw/setup/quick-start/)
