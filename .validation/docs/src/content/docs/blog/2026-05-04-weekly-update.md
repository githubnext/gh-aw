---
title: "Weekly Update – May 4, 2026"
description: "This week brings v0.71.3 with parameterized safe-outputs, the new A/B experiments framework, and a codex harness upgrade."
authors:
  - copilot
date: 2026-05-04
---

Happy May the Fourth! Here's a look at what shipped in [github/gh-aw](https://github.com/github/gh-aw) this week — a busy one packed with experiment infrastructure, compiler fixes, and engine improvements.

## Release: v0.71.3

[v0.71.3](https://github.com/github/gh-aw/releases/tag/v0.71.3) landed on April 30th, capping off a week of rapid iteration. This release delivers major improvements to safe-outputs reusability, more resilient Copilot driver behavior, and solid self-hosted runner support.

### ✨ What's New

- **Parameterized safe-outputs for reusable workflows** ([#29171](https://github.com/github/gh-aw/issues/29171)): `workflow_call` inputs can now control `safe-outputs.threat-detection`, boolean flags, PR policy fields, and list constraints. Build reusable workflows that callers can configure without forking.

- **Configurable MCP gateway session timeout**: Set `engine.mcp.session-timeout` in your workflow frontmatter to keep long-running MCP sessions alive. No more premature timeouts on deep analysis workflows.

- **Auto-inject `create_issue` safe output**: Workflows without explicit safe-output configuration now automatically get a `create_issue` safe output, slashing boilerplate for common workflows.

- **Repo Mind Light shared workflow**: A shared `repo-mind-light.md` workflow is now available for reuse across daily issue/PR agentic workflows ([#29063](https://github.com/github/gh-aw/issues/29063)).

- **Team reviewers on `add_reviewer`**: The `add_reviewer` MCP tool now supports setting `team_reviewers` on pull requests ([#29228](https://github.com/github/gh-aw/issues/29228)).

- **Self-hosted runner support for non-default home directories**: Workflows now work correctly on self-hosted runners where the service account home is not `/home/runner` ([#27260](https://github.com/github/gh-aw/issues/27260)).

## Notable Pull Requests

Several impactful PRs landed this week beyond the release:

- **[Compiler detects single-quoted bash commands that crash Copilot CLI](https://github.com/github/gh-aw/pull/30040)**: The compiler now catches and sanitizes single-quoted bash tool commands before they reach the Copilot CLI, preventing cryptic runtime crashes. A small fix with a big quality-of-life impact.

- **[Default Codex harness with retry logic](https://github.com/github/gh-aw/pull/30035)**: The Codex engine now ships a default `codex_harness.cjs` with built-in retry logic, making Codex-powered workflows more resilient out of the box.

- **[A/B experiments framework](https://github.com/github/gh-aw/pull/30020)**: A hidden `experiments` CLI command lets you read experiment state from storage repo branches, enabling controlled A/B testing of workflow behavior across runs.

- **[Statistical analysis for experiments](https://github.com/github/gh-aw/pull/30029)**: The `experiments analyze` command now computes statistical significance, so you can tell whether a prompt change actually improved things — or just got lucky.

- **[Multiple OTLP endpoints](https://github.com/github/gh-aw/pull/30021)**: The `endpoint` field in OTLP configuration is now polymorphic — send telemetry to multiple backends simultaneously.

- **[Fix: round-robin random start on cache miss](https://github.com/github/gh-aw/pull/30005)**: Round-robin workflows now randomly select their starting item when the cache is cold, preventing all instances from piling onto the first item at startup.

## 🤖 Agent of the Week: ab-testing-advisor

The world's most meta workflow — it finds workflows that *don't* run experiments yet, and proposes experiments for them.

This week `ab-testing-advisor` ran three times, each time scanning the entire workflow catalog for experiment-free candidates, picking one, and writing a detailed GitHub issue with a full A/B experiment campaign. On May 2nd alone it created two issues: one proposing a [`prompt_style` A/B test for the `daily-news` workflow](https://github.com/github/gh-aw/issues/29660) (which it diagnosed as "highly prescriptive" and worth loosening up), and another ([#29661](https://github.com/github/gh-aw/issues/29661)) calling for improvements to the experiment infrastructure itself — the advisor advising on how to improve the advisor. Very on-brand.

It spent roughly 500k tokens per run carefully reading workflow files, thinking through experiment dimensions, and writing crisp implementation specs. For a workflow that runs daily and quietly, it's doing serious intellectual heavy lifting behind the scenes.

💡 **Usage tip**: Use `ab-testing-advisor` as inspiration for your own repos — it's a great example of a meta-workflow that uses AI to drive continuous improvement of *other* AI workflows.

→ [View the workflow on GitHub](https://github.com/github/gh-aw/blob/main/.github/workflows/ab-testing-advisor.md)

## Try It Out

Update to [v0.71.3](https://github.com/github/gh-aw/releases/tag/v0.71.3) today to get parameterized safe-outputs, the new experiment infrastructure, and all the reliability fixes. As always, feedback and contributions are welcome in [github/gh-aw](https://github.com/github/gh-aw).
