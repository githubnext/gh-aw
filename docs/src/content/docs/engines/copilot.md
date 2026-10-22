---
title: Using GitHub Copilot with GitHub Agentic Workflows
description: Authenticate GitHub Copilot and run your first GitHub Agentic Workflow.
---

Run repository automation with your GitHub Copilot account. Copilot is the default engine; you do not need to select it explicitly.

## Connect your account

Grant [`copilot-requests: write`](/gh-aw/reference/auth/#copilot-requests-write-permission) for organization-billed inference, as in the example below. To use a personal Copilot subscription instead, configure [`COPILOT_GITHUB_TOKEN`](/gh-aw/reference/auth/#copilot_github_token).

## Run a daily report

After [setting up gh-aw](/gh-aw/setup/quick-start/), save this workflow:

```aw wrap title=".github/workflows/daily-status.md"
---
on:
  workflow_dispatch:
  schedule: daily
permissions:
  contents: read
  issues: read
  pull-requests: read
  copilot-requests: write
engine: copilot
tools:
  github:
    toolsets: [issues, pull_requests]
safe-outputs:
  create-issue:
    title-prefix: "[status] "
---

# Daily repository status

Summarize open issues and recent pull request activity in one concise issue.
```

Compile with `gh aw compile daily-status`, then commit and push both the Markdown and generated lock file. Run it from the repository's Actions tab or wait for its schedule.

## Capabilities and limitations

For web search, configure an [MCP search server](/gh-aw/reference/web-search/). SDK mode, dynamic workflows, BYOK providers, and custom agents are covered in the [Copilot reference](/gh-aw/reference/engines/copilot/).

## Next steps

- [GitHub Copilot technical reference](/gh-aw/reference/engines/copilot/)
- [Create a workflow](/gh-aw/setup/creating-workflows/)
- [Workflow examples](/gh-aw/gallery/)
- [Compare engines](/gh-aw/reference/engines/#engine-feature-comparison)
