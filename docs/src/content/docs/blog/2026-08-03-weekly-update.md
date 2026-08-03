---
title: "Weekly Update – August 3, 2026"
description: "v0.84.0–v0.84.2 land a shellcheck linting pipeline, safer safe-outputs, container CVE cleanup, and dozens of reliability fixes."
authors:
  - copilot
date: 2026-08-03
metadata:
  seoDescription: "gh-aw ships v0.84.0–v0.84.2 with shellcheck linting, safer safe-outputs, container security fixes, and a busy week of PRs."
---

Another packed week in [github/gh-aw](https://github.com/github/gh-aw): five releases (v0.83.4 through v0.84.2) and over 100 merged pull requests. This week's theme was **hardening** — shell script safety, container security, and closing sneaky edge cases in the safe-outputs pipeline.

## Release Highlights

### [v0.84.2](https://github.com/github/gh-aw/releases/tag/v0.84.2) — August 1

A maintenance release focused on stability and security, with no breaking changes.

- **Fixed an argument injection vulnerability (CWE-88)** in the `git archive` fallback path ([#49500](https://github.com/github/gh-aw/pull/49500))
- **Hardened the PR Description Updater** against one-shot safe-output exhaustion ([#49463](https://github.com/github/gh-aw/pull/49463))
- **Stacked PR runs now default to top-of-stack**, with a configurable `on.pull_request.max-stack` option extended to `pull_request_review` gating ([#49420](https://github.com/github/gh-aw/pull/49420), [#49453](https://github.com/github/gh-aw/pull/49453))
- **Explicit auto-merge strategies** are now supported in `safe-outputs.create-pull-request` ([#49412](https://github.com/github/gh-aw/pull/49412))
- CLI version bumps across the board: Copilot 1.0.77, Pi 0.83.0, Playwright Browser v1.62.1, Syft v1.50.0, Grype v0.116.1 ([#49521](https://github.com/github/gh-aw/pull/49521))

### [v0.84.0 and v0.84.1](https://github.com/github/gh-aw/releases/tag/v0.84.1)

These releases rolled out a **shellcheck linting phase** for generated run steps in the compile pipeline, plus continued security patching for third-party MCP containers.

## Notable Pull Requests

- **[feat: shellcheck disabled by default, opt-in via `--shellcheck`/`--validate`, parallel execution](https://github.com/github/gh-aw/pull/49880)** — the new shellcheck gate ships opt-in first, so teams can adopt it on their own schedule while the compiler still catches real script bugs when enabled.
- **[fix(security): disable semgrep/semgrep container — Critical/High CVEs](https://github.com/github/gh-aw/pull/49694)** and **[security: remove mcp/markitdown container (849 CVEs)](https://github.com/github/gh-aw/pull/49806)** — proactive removal of MCP containers with unpatched vulnerabilities, keeping the default toolset safe by default.
- **[Fix gh-aw-node brace-expansion patch (GHSA-mh99-v99m-4gvg)](https://github.com/github/gh-aw/pull/49853)** — replaced a brittle `npm --prefix` overlay with a temp-dir copy, closing a supply-chain gap.
- **[Add first-class agent job gating via `jobs.agent.needs` and `jobs.agent.if`](https://github.com/github/gh-aw/pull/49814)** — workflow authors can now express fine-grained dependencies and conditions directly on the agent job.
- **[Rescue completed watchdog-fired Copilot runs from false `authentication_failed` classification](https://github.com/github/gh-aw/pull/49792)** — one of several reliability fixes that reduce false-positive failure reports in the dashboards.

## 🤖 Agent of the Week: Dead Code Removal Agent

Every day, this quiet janitor scans the codebase for functions nobody calls anymore — and deletes them, no drama required.

This week it stayed characteristically productive: across its last three scheduled runs it logged zero errors and zero warnings, chewing through roughly 43K tokens total, and its August 1st run ([#49801](https://github.com/github/gh-aw/pull/49801)) walked away with four confirmed dead functions removed in a single pass. One earlier run did hit a rough patch — a merge-conflict-heavy branch tripped it into a "risky" classification — but it shrugged that off and came back clean the very next scheduled run.

It's the kind of agent that never asks for credit: three runs, one clean PR, and a repo that's just a little tidier than it was last Tuesday.

💡 **Usage tip**: Schedule dead-code cleanup agents like this one on a low-traffic cadence (daily or every few days) so PRs stay small, reviewable, and easy to revert if a "dead" function turns out to have a reflection-based caller.

→ [View the workflow on GitHub](https://github.com/github/gh-aw/blob/main/.github/workflows/dead-code-remover.md)

## Try It Out

Update to [v0.84.2](https://github.com/github/gh-aw/releases/tag/v0.84.2) and give the new shellcheck flags a spin with `--validate`. As always, bug reports, security findings, and PRs are welcome in [github/gh-aw](https://github.com/github/gh-aw).
