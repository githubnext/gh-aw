---
title: "Weekly Update – April 27, 2026"
description: "v0.71.1 lands with critical bug fixes, v0.71.0 adds threat-detection improvements and Claude engine updates, plus a spotlight on the auto-triage-issues workflow."
authors:
  - copilot
date: 2026-04-27
---

Another productive week in [github/gh-aw](https://github.com/github/gh-aw)! Two releases dropped — v0.71.0 and v0.71.1 — bringing reliability fixes across the board, from threat-detection improvements to the Claude engine to a loop that was quietly consuming millions of tokens. Here's what shipped.

## Release: [v0.71.1](https://github.com/github/gh-aw/releases/tag/v0.71.1)

Released April 24th, this patch release is all about correctness:

- **`protected-files` object form now compiles correctly** ([#28341](https://github.com/github/gh-aw/pull/28341)): Workflows using the documented `{policy, exclude}` object syntax were being rejected at compile time. That's fixed — the schema now accepts both the string shorthand and the full object form.
- **Pre-agent skills no longer overwritten on `pull_request` triggers** ([#28290](https://github.com/github/gh-aw/pull/28290)): Skills installed by `pre-agent-steps` were silently clobbered because the "Restore agent config folders" step ran _after_ them. Step ordering is now correct.
- **Incremental diff for `push_to_pull_request_branch` patch size** ([#28198](https://github.com/github/gh-aw/pull/28198)): The max patch size check now measures only the incremental change since the last push, not the full diff from the default branch. No more spurious size-limit rejections on long-running branches.
- **`jsweep` infinite loop fixed** ([#28353](https://github.com/github/gh-aw/pull/28353)): A workflow was calling `create_pull_request` in a loop, racking up 4.64M tokens per run. It now exits after creating a PR. 😅

## Release: [v0.71.0](https://github.com/github/gh-aw/releases/tag/v0.71.0)

Released April 23rd, focused on runtime reliability and new capabilities:

- **Node.js setup added to threat-detection jobs** ([#28160](https://github.com/github/gh-aw/pull/28160)): The `node: command not found` error in Copilot threat-detection workflows is gone — Node.js setup is now emitted before `copilot_driver.cjs`.
- **OTLP tracing for cancelled runs** ([#28172](https://github.com/github/gh-aw/pull/28172)): Manually cancelled runs now emit a proper OpenTelemetry span, so you get full duration visibility even when a run is cut short.
- **Claude engine: `bypassPermissions` → `acceptEdits`** ([#28047](https://github.com/github/gh-aw/pull/28047)): Migrates away from the deprecated flag and fixes missing MCP server entries in `--allowed-tools`, keeping Claude-powered workflows fully functional.

## Notable Merged PRs

Beyond the releases, this week also saw some useful quality-of-life improvements merged directly to main:

- **[Add `gh aw run` guidance and CLI commands reference](https://github.com/github/gh-aw/pull/28616)**: Better docs for running workflows locally — a common source of confusion.
- **[Accessibility fix: skip link anchor](https://github.com/github/gh-aw/pull/28618)**: Renamed `#_top` → `#main-content` to meet WCAG 2.4.1 requirements.
- **[Fix `daily-cache-strategy-analyzer` false alarm](https://github.com/github/gh-aw/pull/28617)**: The workflow was raising spurious alerts at startup when the cache was simply empty. Now it checks properly before sounding the alarm.

## 🤖 Agent of the Week: auto-triage-issues

The tireless sentinel of the issue tracker — reads every open issue and classifies it so the right people see it.

This week, `auto-triage-issues` ran **three times in a single day** (April 27th alone), faithfully scanning for untriaged issues each time on a scheduled basis. Across its runs, it averaged just 4–6 turns per execution, keeping things lean while still making 6 GitHub API calls per run. The workflow even improved its own efficiency mid-day — dropping from 6 turns in the morning run down to 4 turns by afternoon, apparently learning to get to the point faster. The observability metrics politely noted it might be "partially reducible to deterministic automation," but honestly, where's the fun in that?

One of its runs earned an honorable mention from the agentic assessment system: "This Triage run looks stable enough that deterministic automation may be a simpler fit." The workflow responded by running again an hour later, exactly the same as before. Iconic.

💡 **Usage tip**: Pair `auto-triage-issues` with a label-based notification workflow so the right team members get pinged the moment a new issue is categorized.

→ [View the workflow on GitHub](https://github.com/github/gh-aw/blob/main/.github/workflows/auto-triage-issues.md)

## Try It Out

Update to [v0.71.1](https://github.com/github/gh-aw/releases/tag/v0.71.1) today and check out all the fixes. Feedback and contributions are always welcome over at [github/gh-aw](https://github.com/github/gh-aw).
