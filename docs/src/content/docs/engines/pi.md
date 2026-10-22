---
title: Using Pi with GitHub Agentic Workflows
description: Authenticate Pi and run your first GitHub Agentic Workflow.
---

Run Pi on GitHub events with GitHub Copilot, Anthropic, OpenAI, or Google inference.

## Connect your account

Use the Copilot configuration below with [`copilot-requests: write`](/gh-aw/reference/auth/#copilot-requests-write-permission), or configure [`COPILOT_GITHUB_TOKEN`](/gh-aw/reference/auth/#copilot_github_token) for personal subscription billing.

For another provider, set `model: anthropic/<model>`, `openai/<model>`, or `google/<model>` and configure its [API key](/gh-aw/reference/auth/). Threat detection uses Copilot by default even when Pi uses another provider; supply Copilot authentication or [select a detection engine](/gh-aw/reference/threat-detection/).

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
engine: pi
model: copilot/auto
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

Pi supports multiple providers, native MCP, and sub-agents. See the [Pi reference](/gh-aw/reference/engines/pi/) for provider routing, tool restrictions, extensions, and SDK/RPC modes.

## Next steps

- [Pi technical reference](/gh-aw/reference/engines/pi/)
- [Create a workflow](/gh-aw/setup/creating-workflows/)
- [Workflow examples](/gh-aw/gallery/)
- [Compare engines](/gh-aw/reference/engines/#engine-feature-comparison)
