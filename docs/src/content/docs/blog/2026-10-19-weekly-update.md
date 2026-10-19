---
title: "Weekly Update – October 19, 2026"
description: "Five v0.92 releases bring structured reports, safer MCP compile, better sub-agent diagnostics and fresh engine upgrades."
authors:
  - copilot
date: 2026-10-19
metadata:
  seoDescription: "This week in gh-aw: v0.92.1 to v0.92.5 add structured reports, read-only MCP compile, sub-agent failure visibility, engine upgrades and security hardening."
---

It was a busy week for [github/gh-aw](https://github.com/github/gh-aw): five pre-releases, [v0.92.1](https://github.com/github/gh-aw/releases/tag/v0.92.1) through [v0.92.5](https://github.com/github/gh-aw/releases/tag/v0.92.5), landed between October 15 and 17. The theme was observability, security hardening and keeping engines current.

## Release Highlights

### [v0.92.5](https://github.com/github/gh-aw/releases/tag/v0.92.5)

A security and diagnostics release.

- **Structured reports** ([#69756](https://github.com/github/gh-aw/pull/69756), [#69755](https://github.com/github/gh-aw/pull/69755)): generated issues now use a normalized report AST, and Actions-native and AW Markdown workflow ASTs were added.
- **Better failure visibility** ([#69788](https://github.com/github/gh-aw/pull/69788), [#69467](https://github.com/github/gh-aw/pull/69467), [#69432](https://github.com/github/gh-aw/pull/69432)): sub-agent failures are surfaced, unavailable models are diagnosed, and `audit`/`logs` handle sub-agent instances without an `agentId`.
- **Safer MCP compile** ([#69510](https://github.com/github/gh-aw/pull/69510)): MCP compile is read-only unless write tools are enabled.
- **Redaction** ([#69520](https://github.com/github/gh-aw/pull/69520), [#69517](https://github.com/github/gh-aw/pull/69517)): detection outputs and CLI proxy startup diagnostics are redacted before publication or artifact upload.
- Dependency bumps to MCP Gateway v0.4.35 and firewall v0.28.62.

### [v0.92.4](https://github.com/github/gh-aw/releases/tag/v0.92.4) and [v0.92.3](https://github.com/github/gh-aw/releases/tag/v0.92.3)

- You can now set `create-issue: false` to disable automatic issue output ([#69369](https://github.com/github/gh-aw/pull/69369)).
- CLI-generated PR descriptions are richer, and there's a new `--pr-message` flag ([#69378](https://github.com/github/gh-aw/pull/69378)).
- `gh aw upgrade` now respects workflow engines and opt-out flags ([#69362](https://github.com/github/gh-aw/pull/69362)).
- Copilot SDK sessions register MCP gateway servers, so MCP tools are available as expected ([#69231](https://github.com/github/gh-aw/pull/69231)).

### [v0.92.2](https://github.com/github/gh-aw/releases/tag/v0.92.2) and [v0.92.1](https://github.com/github/gh-aw/releases/tag/v0.92.1)

- Every `gh aw` command that creates pull requests supports `--auto-merge`.
- Engine upgrades: Copilot CLI 1.0.98, Codex 0.163.0, Gemini CLI 0.64.0, Goose 1.54.0, Kiro CLI 2.31.1 and Antigravity CLI 1.3.6 ([#69146](https://github.com/github/gh-aw/pull/69146)).
- `logs` gained automatic failure clustering and investigation plans ([#68954](https://github.com/github/gh-aw/pull/68954)).
- New offline skill command and SAML-safe instruction downloads ([#69043](https://github.com/github/gh-aw/pull/69043)).
- `health` supports logs-style date windows ([#69048](https://github.com/github/gh-aw/pull/69048)).

## 🤖 Agent of the Week: daily-firewall-report

The network watchdog that collects firewall logs and reports on which domains your agentic workflows are reaching.

Its three most recent runs (October 6–8) were scheduled early-morning runs on the Copilot engine. Each ran for 10 to 27 minutes, and the firewall data it gathered showed 81 requests across the three runs, all allowed and none blocked, spread across `api.githubcopilot.com`, Sentry and Grafana ingest endpoints.

The funny part: a firewall reporter whose own firewall had nothing to block, while the runs themselves all stopped at the MCP Gateway start step. Even the watchers need a good night's sleep.

💡 **Usage tip**: Schedule a firewall report so unexpected domains show up in a daily summary instead of during an incident.

→ [View the workflow on GitHub](https://github.com/github/gh-aw/blob/main/.github/workflows/daily-firewall-report.md)

## Try It Out

Upgrade to [v0.92.5](https://github.com/github/gh-aw/releases/tag/v0.92.5) and tell us what you think. Feedback and contributions are always welcome in [github/gh-aw](https://github.com/github/gh-aw).
