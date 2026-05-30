---
title: "Weekly Update – April 13, 2026"
description: "Five releases this week: engine.bare context control, a critical Copilot CLI hotfix, cross-job distributed tracing, and a wave of security hardening."
authors:
  - copilot
date: 2026-04-13
---

It was a busy week in [github/gh-aw](https://github.com/github/gh-aw) — five releases shipped between April 6 and April 10, addressing everything from a critical Copilot CLI reliability crisis to shiny new workflow composition features. Here's the full rundown.

## Release Highlights

### [v0.68.1](https://github.com/github/gh-aw/releases/tag/v0.68.1) — April 10

The headline of this patch is a **critical Copilot CLI reliability hotfix**. Workflows using the Copilot engine were hanging indefinitely or producing zero-byte output due to an incompatibility introduced in v1.0.22 of the Copilot CLI. [v0.68.1](https://github.com/github/gh-aw/releases/tag/v0.68.1) pins the CLI back to v1.0.21 — the last confirmed-working version — and gets everyone's workflows running again ([#25689](https://github.com/github/gh-aw/pull/25689)).

Beyond the hotfix, this release also ships:

- **`engine.bare` frontmatter field** ([#25661](https://github.com/github/gh-aw/pull/25661)): Set `bare: true` to suppress automatic context loading — `AGENTS.md` and user instructions for Copilot, `CLAUDE.md` memory files for Claude. Great when you want the AI to start from a clean slate.
- **Improved stale lock file diagnostics** ([#25571](https://github.com/github/gh-aw/pull/25571)): When the activation job detects a stale hash, it now emits step-by-step `[hash-debug]` log lines and opens an actionable issue guiding you to fix it.
- **`actions/github-script` upgraded to v9** ([#25553](https://github.com/github/gh-aw/pull/25553)): Scripts now get `getOctokit` as a built-in context parameter, removing the need for manual `@actions/github` imports in safe-output handlers.
- **Squash-merge fallback in `gh aw add`** ([#25609](https://github.com/github/gh-aw/pull/25609)): If a repo disallows merge commits, the setup PR now automatically falls back to squash merge instead of failing.
- **Security: `agent-stdio.log` permissions hardened** — Log files are now pre-created with `0600` permissions before `tee` writes, preventing world-readable exposure of MCP gateway bearer tokens.

### [v0.68.0](https://github.com/github/gh-aw/releases/tag/v0.68.0) — April 10

This release brings [distributed tracing](https://github.com/github/gh-aw/releases/tag/v0.68.0) improvements and a cleaner comment API:

- **OpenTelemetry cross-job trace hierarchy** ([#25540](https://github.com/github/gh-aw/pull/25540)): Parent span IDs now propagate through `aw_context` across jobs, giving you end-to-end distributed trace visibility for multi-job workflows in backends like Tempo, Honeycomb, and Datadog.
- **Simplified discussion comment API** ([#25532](https://github.com/github/gh-aw/pull/25532)): The deprecated `add-comment.discussion` boolean has been removed in favor of the clearer `discussions: true/false` syntax. Run `gh aw fix --write` to migrate existing workflows.
- **Security: heredoc content validation** ([#25510](https://github.com/github/gh-aw/pull/25510)): `ValidateHeredocContent` checks now cover five user-controlled heredoc insertion sites, closing a class of potential injection vectors.

### [v0.67.4](https://github.com/github/gh-aw/releases/tag/v0.67.4) — April 9

This one led with **five new agentic workflow templates**: [approach-validator](https://github.com/github/gh-aw/pull/25354), [test-quality-sentinel](https://github.com/github/gh-aw/pull/25353), [refactoring-cadence](https://github.com/github/gh-aw/pull/25352), [architecture-guardian](https://github.com/github/gh-aw/pull/25334), and [design-decision-gate](https://github.com/github/gh-aw/pull/25323). These expand the built-in library for code quality, ADR enforcement, and architectural governance. The release also included Copilot driver retry logic and a `--runner-guard` compilation flag.

### [v0.67.3](https://github.com/github/gh-aw/releases/tag/v0.67.3) — April 8

The star of this release is the new **`pre-steps` frontmatter field** — inject steps that run _before_ checkout and the agent inside the same job. This is the recommended pattern for token-minting actions (e.g., `actions/create-github-app-token`, `octo-sts`) that need to check out external repos. Because the minted token stays in the same job, it never gets masked when crossing a job boundary. Also shipped: `${{ github.aw.import-inputs.* }}` expression support in the `imports:` section, and `assignees` support on `create-pull-request` fallback issues.

### [v0.67.2](https://github.com/github/gh-aw/releases/tag/v0.67.2) — April 6

Reliability-focused: cross-repo workflow hash checks, checkout tokens no longer silently dropped on newer runners, `curl`/`wget` flag-bearing invocations now allowed in `network.allowed` workflows, and a `timeout-minutes` schema cap at 360.

## Notable Merged Pull Requests

Beyond the releases, the past week also delivered:

- **[#25923](https://github.com/github/gh-aw/pull/25923)**: Image artifacts can now be uploaded without zip archiving using `skip-archive: true`, and the resulting artifact URLs are surfaced as outputs — enabling workflows to embed images directly in Markdown comments.
- **[#25908](https://github.com/github/gh-aw/pull/25908)**: A new scheduled `cleanup-cache-memory` job was added to the agentics maintenance workflow to prune outdated cache-memory entries automatically (and can be triggered on demand).
- **[#25914](https://github.com/github/gh-aw/pull/25914) + [#25972](https://github.com/github/gh-aw/pull/25972)**: OTel exception span events now emit `exception.type` alongside `exception.message` and individual error attributes are queryable — no more digging through pipe-delimited strings in Grafana.
- **[#25960](https://github.com/github/gh-aw/pull/25960)**: Fixed a sneaky bug where `push_repo_memory` would run on every bot-triggered no-op because `always()` bypassed skip propagation.
- **[#25971](https://github.com/github/gh-aw/pull/25971)**: Raw subprocess output from `gh aw compile --validate` is now sanitized before being embedded into issue bodies, closing a Markdown injection vector.

## 🤖 Agent of the Week: auto-triage-issues

The quiet backbone of issue hygiene — reads every new issue and applies the right labels so the right people see it.

This week `auto-triage-issues` proved it's doing its job almost too well. In the scheduled run on April 13, it scanned all open issues and found exactly **zero** unlabeled issues — reporting a 100% label coverage rate with zero action required. It had already handled the labeling in near-real-time as issues arrived, including one run on April 12 where it correctly tagged a freshly opened issue with `enhancement`, `mcp`, `compiler`, and `security` in a single pass. Four labels, zero hesitation.

That "security" label is doing a lot of work — the workflow spotted MCP and compiler concerns that genuinely deserved the tag, not just keyword-matched on it. We'll take it.

💡 **Usage tip**: Pair `auto-triage-issues` with label-based notification rules so your team gets automatically paged for `security` or `critical` issues without anyone having to babysit the issue tracker.

→ [View the workflow on GitHub](https://github.com/github/gh-aw/blob/main/.github/workflows/auto-triage-issues.md)

## Try It Out

Update to [v0.68.1](https://github.com/github/gh-aw/releases/tag/v0.68.1) today to get the Copilot CLI hotfix and the new `engine.bare` control. As always, contributions and feedback are welcome in [github/gh-aw](https://github.com/github/gh-aw).
