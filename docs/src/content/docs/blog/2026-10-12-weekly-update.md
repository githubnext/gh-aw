---
title: "Weekly Update – October 12, 2026"
description: "Five releases (v0.91.4–v0.91.8) bring model routing, an experimental Agy engine, Git-backed work queues, and unified session timelines."
authors:
  - copilot
date: 2026-10-12
metadata:
  seoDescription: "gh-aw v0.91.4–v0.91.8: model routing and fallback, the Agy engine, Git-backed work queues, unified session timelines, and security hardening."
---

Five releases shipped in [github/gh-aw](https://github.com/github/gh-aw) this week, from [v0.91.4](https://github.com/github/gh-aw/releases/tag/v0.91.4) to [v0.91.8](https://github.com/github/gh-aw/releases/tag/v0.91.8). The themes: smarter model routing, new engine options, and sturdier sandboxes.

## Release Highlights

### [v0.91.8](https://github.com/github/gh-aw/releases/tag/v0.91.8)

Agent sessions now render as full elapsed-time timelines, with consistent trace normalization across Copilot, Codex, Claude, Gemini, Pi, Kiro, Aider and AGY.

- **Built-in graders**: a new `code-read-precision` trajectory grader and builtin grader defaults loaded from setup-action JSON.
- **Model routing**: Copilot wire APIs are selected from AWF metadata and model mismatches are surfaced. The firewall moved to v0.28.51.
- **Hardening**: a custom grader sandbox escape was closed, and workflow YAML, scanner file handling and temp-file writes were tightened.
- **Memory validation**: declarative JSON schema validation for memory, with repairable invalid repo-memory baselines.

### [v0.91.7](https://github.com/github/gh-aw/releases/tag/v0.91.7)

This release introduces a native Go Agy engine ([#66998](https://github.com/github/gh-aw/pull/66998)) and deprecates the Gemini engine in its favor ([#66941](https://github.com/github/gh-aw/pull/66941)), so plan to migrate Gemini workflows.

- **Work queue upgrades**: priority-aware conformance dispatch ([#67222](https://github.com/github/gh-aw/pull/67222)) and scoped GitHub Issues mirrors ([#67025](https://github.com/github/gh-aw/pull/67025)).
- **Better model visibility**: routed model and effort are reported across outputs ([#66963](https://github.com/github/gh-aw/pull/66963)), plus per-agent spend ([#66961](https://github.com/github/gh-aw/pull/66961)).
- **Secret detection** now covers all AWF environment sources ([#67017](https://github.com/github/gh-aw/pull/67017)).

### [v0.91.6](https://github.com/github/gh-aw/releases/tag/v0.91.6)

- **Git-backed fair DAG work queues** with claim-scoped effects ([#66024](https://github.com/github/gh-aw/pull/66024)).
- **Model routing** across Claude, Codex, and pi ([#66723](https://github.com/github/gh-aw/pull/66723)).
- **Stricter `pull_request_target` checkouts** via an explicit strict-mode policy ([#66649](https://github.com/github/gh-aw/pull/66649)).

### [v0.91.5](https://github.com/github/gh-aw/releases/tag/v0.91.5)

- **Ordered engine model fallback** keeps workflows running when a preferred model is unavailable ([#66593](https://github.com/github/gh-aw/pull/66593)).
- **`auto` and `copilot/auto` models** now work in Pi, Codex, and Claude ([#66567](https://github.com/github/gh-aw/pull/66567)).
- **New `gh aw fix` codemods** for Copilot web-fetch and reserved inputs ([#66556](https://github.com/github/gh-aw/pull/66556)).

### [v0.91.4](https://github.com/github/gh-aw/releases/tag/v0.91.4)

[v0.91.4](https://github.com/github/gh-aw/releases/tag/v0.91.4) adds support for [Claude Code with GitHub Copilot inference](https://github.com/github/gh-aw/pull/66289) and [local-first workflow debugging](https://github.com/github/gh-aw/pull/66270).

## 🤖 Agent of the Week: unbloat-docs

Documentation Unbloat reviews the docs and simplifies verbose pages while keeping them clear and complete.

This week it ran every night on its schedule at about 01:15 UTC. All three runs we looked at (October 10, 11 and 12) succeeded, taking between 7 and 10 minutes and producing one safe output each.

Its stated mission is to make docs shorter, so it spends its nights trimming words, which is a bit like a barber who works only while you sleep.

💡 **Usage tip**: Pair a scheduled editor like this with a `/unbloat` slash command so you can also trim a specific page on demand.

→ [View the workflow on GitHub](https://github.com/github/gh-aw/blob/main/.github/workflows/unbloat-docs.md)

## Try It Out

Upgrade to [v0.91.8](https://github.com/github/gh-aw/releases/tag/v0.91.8) and see the [full release list](https://github.com/github/gh-aw/releases). Feedback and contributions are welcome in [github/gh-aw](https://github.com/github/gh-aw).
