---
title: "Weekly Update – June 29, 2026"
description: "This week in gh-aw: a brand-new Copilot Canvas extension, sandbox hardening hits 80%, Code Scanning Fixer expands to all severity levels, and more."
authors:
  - copilot
date: 2026-06-29
metadata:
  seoDescription: "gh-aw weekly update: Copilot Canvas extension for agentic workflows, 80% sandbox hardening, Code Scanning Fixer for all severities, mcpg v0.3.32."
---

A big week at [github/gh-aw](https://github.com/github/gh-aw)! Canvas extensions land, security coverage expands, and the runtime stack gets a fresh set of bumps. Here's everything that shipped between June 22 and June 29.

## 🖼️ New: Copilot Canvas Extension for Agentic Workflows

[PR #42137](https://github.com/github/gh-aw/pull/42137) ships a project-scoped GitHub Copilot Canvas extension — a GitHub-styled dashboard you can open right inside the Copilot app to manage agentic workflows without leaving your editor.

The extension supports:

- **Browse definitions and runs** — `listDefinitions(page, pageSize)` and `listRuns(page, pageSize)` with full pagination
- **Inspect runs** — `getRun(id)` returns rich step summaries with safe markdown rendering
- **Dispatch workflows** — kick off any workflow via `dispatchWorkflow(definitionId, inputs)`
- **Run CLI commands in-canvas** — `runGhAwLogs(args)` and `runGhAwAudit(args)` bring `gh aw logs` and `gh aw audit` into the canvas surface

The UI is built with Alpine.js and Primer CSS using native ES modules, with strict TypeScript domain models (`WorkflowDefinition`, `WorkflowRun`, `WorkflowStep`) and deterministic in-memory pagination. This is a high-impact addition for anyone who manages agentic workflows day-to-day.

Paired with this, [PR #42147](https://github.com/github/gh-aw/pull/42147) adds a new **`create-canvas` skill** that guides you through authoring, validating, and debugging canvas extensions — covering the full lifecycle from scaffolding via `extensions_manage` to exercising actions with `invoke_canvas_action`.

## 🔒 Security: Sandbox Hardening Reaches 80%

[PR #42119](https://github.com/github/gh-aw/pull/42119) is a satisfying milestone: `sandbox.agent.sudo: false` is now set on **206 out of 257 workflows (80.16%)**. This PR added the flag to 79 additional workflow specs and regenerated the matching lock files. Provenance-managed (`source:`) workflows were left untouched. If your workflow audits were catching a lot of missing sandbox flags, this cleans up the bulk of them.

## 🛡️ Code Scanning Fixer: Now Covers All Severity Levels

Previously, [code-scanning-fixer](https://github.com/github/gh-aw/blob/main/.github/workflows/code-scanning-fixer.md) only tackled `critical` and `high` alerts. [PR #42139](https://github.com/github/gh-aw/pull/42139) removes that filter, expanding the workflow to enumerate **all open code scanning alerts** and prioritize them by severity:

1. `critical > high > medium > low` (using `rule.security_severity_level` when available)
2. Falls back to `error > warning > note` when security severity is absent

The selection logic, no-op messaging, and PR body copy were all generalized to work across every severity level. If you had a backlog of medium/low findings quietly aging, this workflow will now start chipping away at them.

## 📦 Runtime: mcpg v0.3.32 + Firewall v0.27.13

[PR #42146](https://github.com/github/gh-aw/pull/42146) bumps two default runtime components:

| Component | Old | New |
|---|---|---|
| gh-aw-mcpg | v0.3.31 | v0.3.32 |
| gh-aw-firewall | v0.27.12 | v0.27.13 |

All container image digests are SHA-pinned in `action_pins.json`. Low-risk, no migration needed.

## 🔧 Other Merges Worth Noting

- [PR #42115](https://github.com/github/gh-aw/pull/42115) — linter-miner added a new `osgetenvlibrary` analyzer that flags `os.Getenv`/`LookupEnv` calls in library packages (environment coupling in libraries is a common footgun).
- [PR #42118](https://github.com/github/gh-aw/pull/42118) — Prevents step-summary conversation truncation when agent output contains fenced code blocks.
- [PR #42117](https://github.com/github/gh-aw/pull/42117) — Slash-command footer hints now render correctly for custom safe-output footers.
- [PR #42112](https://github.com/github/gh-aw/pull/42112) — Fixed a cache-memory history path bug in Agent Persona Explorer that was triggering false `cache_memory_miss` errors.

---

## 🤖 Agent of the Week: agent-persona-explorer

A research agent that turns the lens inward — it systematically tests the `agentic-workflows` custom agent by roleplaying as different worker personas and evaluating what comes back.

Each run, `agent-persona-explorer` picks three personas from a pool of nine — Backend Engineer, Frontend Developer, DevOps Engineer, Data Scientist, Product Manager, and more — generates 2 automation scenarios per persona, then submits each to the `agentic-workflows` agent and scores the responses on five dimensions: clarity, tool selection, security awareness, efficiency, and output quality. It stores a rotation history in cache memory so it never tests the same persona slice twice in a row. Results are published as a GitHub issue labeled `agent-research`.

The workflow ran three times in the past week. Two runs succeeded cleanly, but the first failed due to a cache-memory path mismatch — which was fixed in [PR #42112](https://github.com/github/gh-aw/pull/42112) within hours. The two successful runs consumed around 24 AIC each, used `gpt-5.4` for analysis, and made 13 GitHub API calls to gather workflow context before synthesizing findings.

There's also a quiet A/B experiment running in the background (since May 2026): the workflow is testing whether batching all persona scenarios into one sub-agent call is cheaper than spawning a separate sub-agent per scenario. The hypothesis is a ≥20% token reduction — and with 14 minimum samples required for a t-test conclusion, the jury is still out.

💡 **Usage tip**: If you're building or tuning a custom agent, `agent-persona-explorer`-style testing is a powerful way to surface blind spots — run it against your own agent to see how it handles requests from personas you didn't design for.

→ [View the workflow on GitHub](https://github.com/github/gh-aw/blob/main/.github/workflows/agent-persona-explorer.md)

---

Check out the [github/gh-aw](https://github.com/github/gh-aw) repository for the full list of changes, and give the new Canvas extension a spin if you're managing agentic workflows in Copilot.
