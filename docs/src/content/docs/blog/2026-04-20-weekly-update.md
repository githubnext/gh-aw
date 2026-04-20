---
title: "Weekly Update – April 20, 2026"
description: "This week brings five releases packed with a new OpenCode engine, pre-agent steps, cache-memory security hardening, and much more."
authors:
  - copilot
date: 2026-04-20
---

What a week for [github/gh-aw](https://github.com/github/gh-aw)! Five releases dropped between April 13 and April 17, delivering a new AI engine, key security improvements, and a wave of reliability fixes. Here's what you need to know.

## Release Highlights

### [v0.68.7](https://github.com/github/gh-aw/releases/tag/v0.68.7) — April 17

A targeted fix-and-polish release with one standout new addition:

- **`on.roles` single-string support** ([#26789](https://github.com/github/gh-aw/pull/26789)): You can now write `roles: write` instead of `roles: [write]`. Previously this produced a confusing compiler error — now it just works.
- **Codex chroot fix** ([#26787](https://github.com/github/gh-aw/pull/26787)): Codex workflows on restricted filesystems were failing silently. Runtime state now lives in `/tmp` where it can actually be written.
- **Cross-repo compatibility checks** ([#26802](https://github.com/github/gh-aw/pull/26802)): A new daily Claude workflow automatically discovers repositories using gh-aw and runs compile checks against the latest build. Compatibility regressions now get caught before they reach users.

### [v0.68.6](https://github.com/github/gh-aw/releases/tag/v0.68.6) — April 17

The headline release of the week, with a brand-new engine and important security improvements:

- **OpenCode engine** — Set `engine: opencode` to use [OpenCode](https://opencode.ai) as your agentic engine, joining Copilot, Claude, and Codex as first-class options.
- **`engine.bare` mode** — Set `engine.bare: true` to skip loading `AGENTS.md`. Perfect for triage, reporting, and ops workflows where repository code context just adds noise.
- **Pre-agent steps** — The new `pre-agent-steps` frontmatter field lets you run custom GitHub Actions steps before the AI agent starts — great for authentication, environment setup, or any prerequisite work.
- **`cache-memory` working-tree sanitization** — Before each agent run, the working tree is now scanned and cleaned of planted executables and disallowed files from cached memory. This closes a real supply-chain attack vector.

### [v0.68.5](https://github.com/github/gh-aw/releases/tag/v0.68.5) — April 16

Quality-of-life improvements and more security hardening:

- **MCP config at `.github/mcp.json`** ([#26665](https://github.com/github/gh-aw/pull/26665)): The MCP configuration file has moved from `.mcp.json` (repo root) to `.github/mcp.json`, aligning with standard GitHub configuration conventions. The `init` flow creates the new path automatically.
- **`shared/reporting-otlp.md` import bundle** ([#26655](https://github.com/github/gh-aw/pull/26655)): One import now replaces two for telemetry-enabled reporting workflows.
- **Environment-level secrets fixed** ([#26650](https://github.com/github/gh-aw/pull/26650)): The `environment:` frontmatter field now correctly propagates to the activation job.

### [v0.68.4](https://github.com/github/gh-aw/releases/tag/v0.68.4) — April 16

A substantial patch resolving 21 community-reported issues:

- **BYOK Copilot mode** ([#26544](https://github.com/github/gh-aw/pull/26544)): New `byok-copilot` feature flag wires offline Copilot support.
- **SideRepoOps maintenance workflow** ([#26382](https://github.com/github/gh-aw/pull/26382)): The compiler now auto-generates `agentics-maintenance.yml` for target repositories in SideRepoOps patterns.
- **MCP servers as local CLIs** ([#25928](https://github.com/github/gh-aw/pull/25928)): MCP servers can now be mounted as local CLI commands after the gateway starts, enabling richer tool integrations.

### [v0.68.3](https://github.com/github/gh-aw/releases/tag/v0.68.3) — April 14

Observability and reliability improvements:

- **Model-not-supported detection** ([#26229](https://github.com/github/gh-aw/pull/26229)): When a model is unavailable for your plan, the workflow now stops retrying and surfaces a clear error instead of spinning indefinitely.
- **Time Between Turns (TBT) metric** ([#26321](https://github.com/github/gh-aw/pull/26321)): `gh aw audit` and `gh aw logs` now report TBT — a key indicator of whether LLM prompt caching is working for your workflows.
- **`env` and `checkout` fields in shared imports** ([#26113](https://github.com/github/gh-aw/pull/26113), [#26292](https://github.com/github/gh-aw/pull/26292)): Shared importable workflows now support both `env:` and `checkout:` fields, eliminating common workarounds.

## 🤖 Agent of the Week: auto-triage-issues

The unsung hero of issue hygiene — reads every unlabeled issue and applies the right labels so the right people see it, automatically, on a schedule.

This week `auto-triage-issues` kept its usual steady pace, triaging issues as they came in. In one run, it spotted issue [#27290](https://github.com/github/gh-aw/issues/27290) — a question about ecosystem groups in the frontmatter/compilation pipeline — and correctly labeled it `compiler` within 24 seconds flat. In another run, it encountered an issue that the integrity policy had filtered before the agent could even read the title, so it did the responsible thing: skipped labeling, created a summary discussion, and politely told the maintainers to take a look themselves.

Even when it can't act, it doesn't just silently fail — it leaves a breadcrumb so nothing falls through the cracks.

💡 **Usage tip**: Pair `auto-triage-issues` with a `notify` workflow on high-priority labels (like `security` or `breaking-change`) so your team gets paged for the things that actually matter.

→ [View the workflow on GitHub](https://github.com/github/gh-aw/blob/main/.github/workflows/auto-triage-issues.md)

## Try It Out

With [v0.68.7](https://github.com/github/gh-aw/releases/tag/v0.68.7) now available, it's a great time to update and explore the new OpenCode engine, `engine.bare` mode, or pre-agent steps. As always, feedback and contributions are very welcome in [github/gh-aw](https://github.com/github/gh-aw).
