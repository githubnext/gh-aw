---
title: "Weekly Update – August 10, 2026"
description: "This week brought v0.86.1 with guided gh aw fix diagnostics, expanded engine support, and a heavy security hardening pass in v0.86.0."
authors:
  - copilot
date: 2026-08-10
metadata:
  seoDescription: "gh-aw weekly update: v0.86.1 diagnostics, v0.86.0 security hardening, PureLock test coverage, and dozens of merged PRs."
---

It's been another busy week in [github/gh-aw](https://github.com/github/gh-aw), with two notable releases and dozens of merged pull requests touching everything from compiler safety to CI stability. Here's what shipped.

## Release: v0.86.1

[v0.86.1](https://github.com/github/gh-aw/releases/tag/v0.86.1) landed on August 7th with a broad set of compiler safety fixes, new `gh aw fix` diagnostics, and expanded engine support.

### What's New

- **Guided `gh aw fix` diagnostics**: The tool now offers a guided fix for restricted `tools.bash` allow-listing on engines that ignore it ([#51102](https://github.com/github/gh-aw/pull/51102)), plus tips for known external engines like opencode and crush missing their import ([#51088](https://github.com/github/gh-aw/pull/51088)).
- **Expanded engine support**: Added new example workflows for the aider, cursor, and kiro definition-based engines ([#51166](https://github.com/github/gh-aw/pull/51166)).
- **PureLock initiative**: Introduced a daily pure-function maximum-coverage test workflow ([#51107](https://github.com/github/gh-aw/pull/51107)) that is progressively locking down core compiler functions with dedicated test suites ([#51167](https://github.com/github/gh-aw/pull/51167), [#51119](https://github.com/github/gh-aw/pull/51119)).
- **Safe-outputs improvements**: Fixed `add_labels` failing on pull requests in issue-intent paths ([#51168](https://github.com/github/gh-aw/pull/51168)) and replaced loosely-typed bool-or-expression fields with `*TemplatableBool` for safer config typing ([#51097](https://github.com/github/gh-aw/pull/51097)).

## Release: v0.86.0

[v0.86.0](https://github.com/github/gh-aw/releases/tag/v0.86.0) shipped earlier the same day as a heavy security and reliability hardening pass across secret redaction, MCP gateway logging, and threat-detection resilience.

### Security & Redaction

- **Secrets can no longer leak through logs or artifacts.** Redaction is now enforced in step summaries ([#50777](https://github.com/github/gh-aw/pull/50777)), patch/bundle artifacts ([#50778](https://github.com/github/gh-aw/pull/50778)), and MCP gateway diagnostic logs ([#50961](https://github.com/github/gh-aw/pull/50961)).
- **URL handling hardened**: userinfo is now stripped from logged URLs and rejected URLs are no longer logged in full ([#50776](https://github.com/github/gh-aw/pull/50776)).
- **`upload_artifact` safe-output** now restricts uploads to canonical allowed roots and rejects sensitive paths ([#50779](https://github.com/github/gh-aw/pull/50779)).

## Notable Pull Requests

Beyond the releases, the team merged a steady stream of fixes and quality-of-life improvements:

- [Skip stale review-thread node IDs instead of failing safe_outputs](https://github.com/github/gh-aw/pull/51630) — makes `resolve_pull_request_review_thread` more resilient to stale GraphQL node IDs.
- [Grant agentic engines read/write access to /tmp/gh-aw in AWF sandbox](https://github.com/github/gh-aw/pull/51608) — smooths out sandboxed engine runs that need scratch space.
- [Fix recurring gh-aw-firewall digest-pin loss on DefaultFirewallVersion bumps](https://github.com/github/gh-aw/pull/51423) — keeps firewall image pins from silently drifting on version bumps.
- [Add explicit end marker syntax for inline skills and sub-agents](https://github.com/github/gh-aw/pull/51446) — clarifies where inline skill and sub-agent content ends in workflow markdown.
- [Fix Copilot path portability across runners](https://github.com/github/gh-aw/pull/51275) — irons out cross-platform path handling for the Copilot engine.

## 🤖 Agent of the Week: PureLock

PureLock is the daily workflow that quietly locks down up to three uncovered pure Go functions per run, writing dedicated test suites so core compiler logic doesn't regress unnoticed.

This week PureLock ran three times — once from its daily schedule and twice via manual dispatch — clocking in at 15 to 22 minutes per run and burning through roughly 60,000 tokens total. All three runs completed successfully and stayed strictly read-only until their final PR, methodically chipping away at coverage gaps. Its handiwork showed up directly in this week's release notes, with [#51586](https://github.com/github/gh-aw/pull/51586) locking down `sameExpr`, `addAllowedToNetwork`, and `rpcEntryToTimelineEvent` with pure-function test suites.

Give it a function name like `simplifyDataSchemaNode` and it will happily go write exhaustive tests for it without complaint — the kind of unglamorous, repetitive work that keeps a growing Go codebase honest one pure function at a time.

💡 **Usage tip**: Pair a coverage-locking workflow like this with your CI's coverage gate so newly written tests actually prevent regressions instead of just padding a report.

→ [View the workflow on GitHub](https://github.com/github/gh-aw/blob/main/.github/workflows/purelock.md)

## Try It Out

Update to [v0.86.1](https://github.com/github/gh-aw/releases/tag/v0.86.1) today to get the latest diagnostics and security hardening. As always, feedback and contributions are welcome in [github/gh-aw](https://github.com/github/gh-aw).
