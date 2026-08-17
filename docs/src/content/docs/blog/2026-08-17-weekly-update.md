---
title: "Weekly Update – August 17, 2026"
description: "v0.87.0 ships fork PR approval safe outputs, cloud-hypervisor isolation, and a big internal hardening pass."
authors:
  - copilot
date: 2026-08-17
metadata:
  seoDescription: "gh-aw v0.87.0 adds an approve-workflow-run safe output, cloud-hypervisor isolation, new lint rules, and dozens of reliability fixes."
---

Another busy week for [github/gh-aw](https://github.com/github/gh-aw)! The team shipped [v0.87.0](https://github.com/github/gh-aw/releases/tag/v0.87.0), a release packed with security hardening, a new safe output for fork pull requests, and dozens of smaller reliability improvements across the workflow ecosystem.

## Release: v0.87.0

[v0.87.0](https://github.com/github/gh-aw/releases/tag/v0.87.0) landed on August 16th and is best described as a major internal hardening pass — custom linters, security fixes, and refactors — plus one notable new capability.

### What's New

- **Approve fork pull request workflow runs** ([#52541](https://github.com/github/gh-aw/pull/52541)): a new experimental `approve-workflow-run` safe output lets agents programmatically unblock GitHub's fork PR approval gate, with strict guardrails including protected-file checks, allowed-workflow/PR scoping, and required external tokens. See [ADR-52541](https://github.com/github/gh-aw/blob/main/docs/adr/52541-add-approve-workflow-run-safe-output.md) for the design rationale.
- **Cloud-hypervisor agent runtime** ([#52932](https://github.com/github/gh-aw/pull/52932)): enabled on eligible agentic workflows for improved isolation.
- **Model inventory refresh** ([#52993](https://github.com/github/gh-aw/pull/52993)): added Gemini 3.7 Flash and Grok 4.6 to the supported model list.
- **Extended confused-deputy protection** ([#52976](https://github.com/github/gh-aw/pull/52976)) to `pull_request_target` triggers, plus removal of several vulnerable/deprecated container image pins (`gh-aw-firewall`, `cli-proxy`, Serena MCP).

Full details, including the dozens of new custom lint rules that shipped alongside this release, are in the [v0.87.0 release notes](https://github.com/github/gh-aw/releases/tag/v0.87.0).

## Notable Pull Requests

Beyond the release, the past week saw a steady stream of merged fixes and improvements:

- [Migrate 30 more agentic workflows to the gh-aw-detection feature](https://github.com/github/gh-aw/pull/53162) — continues the rollout of the shared detection framework across more of the repository's workflows.
- [Fix Aider engine producing no safe outputs by pinning the diff edit format](https://github.com/github/gh-aw/pull/53160) — resolves a silent failure mode where the Aider engine could complete a run without emitting any safe outputs.
- [Harden sandbox.mcp.env key encoding at emission and launch boundaries](https://github.com/github/gh-aw/pull/53171) — tightens how MCP environment variables are encoded to avoid injection risks.
- [Fill documentation gaps found by Agent Persona Explorer run](https://github.com/github/gh-aw/pull/53155) — adds missing docs for schedule-based compliance audits, PM digests, and multi-scenario comparisons, surfaced by an internal agent persona exploration run.
- [Handle bare Design Decision Gate workflow dispatch](https://github.com/github/gh-aw/pull/53202) — fixes an edge case in manual dispatch handling for the Design Decision Gate workflow.

## 🤖 Agent of the Week: Issue Arborist

Every night at the crack of dawn (05:38 UTC, to be exact), [Issue Arborist](https://github.com/github/gh-aw/blob/main/.github/workflows/issue-arborist.md) quietly analyzes the repository's recent issues and links related ones together as sub-issues — building out the "family tree" of the issue tracker one branch at a time.

Over its last three scheduled runs, Issue Arborist has been remarkably consistent: each run completes in under 11 minutes, churns through around 15 GitHub API calls, and reliably produces 8–14 safe output items per run — all without a single error or missing tool across the board. It's the kind of quiet, dependable background work that keeps the issue tracker organized without anyone having to lift a finger.

There's something charmingly patient about a workflow whose entire job is to notice "hey, these two issues are actually talking about the same thing" — night after night, without complaint, and without ever needing a coffee break.

💡 **Usage tip**: Schedule-based triage workflows like this one work best when paired with `skip-if-match` conditions (as Issue Arborist uses) to avoid redundant runs when there's nothing new to organize.

→ [View the workflow on GitHub](https://github.com/github/gh-aw/blob/main/.github/workflows/issue-arborist.md)

## Try It Out

Check out [v0.87.0](https://github.com/github/gh-aw/releases/tag/v0.87.0) and give the new `approve-workflow-run` safe output a spin if you work with fork pull requests. As always, feedback and contributions are welcome over at [github/gh-aw](https://github.com/github/gh-aw).
