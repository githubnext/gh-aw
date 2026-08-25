---
title: "Weekly Update – August 24, 2026"
description: "This week brought three v0.87.x pre-releases with stricter safe-outputs validation, new pre-create PR steering, and a wave of internal reliability fixes."
authors:
  - copilot
date: 2026-08-24
metadata:
  seoDescription: "gh-aw weekly update: v0.87.x releases, pre-create PR steering, and reliability fixes across the agentic workflow pipeline."
---

Another busy week for [github/gh-aw](https://github.com/github/gh-aw)! We shipped three pre-releases (`v0.87.1`, `v0.87.2`, and `v0.87.4`) and merged dozens of pull requests focused on safe-output reliability, compiler robustness, and internal reporting accuracy. Here's what stood out.

## Release Highlights

The [v0.87.4](https://github.com/github/gh-aw/releases/tag/v0.87.4) line of releases focused on compiler robustness, safe-output validation, and internal tooling and observability improvements across the agentic workflow pipeline.

- **Run steering** ([#55171](https://github.com/github/gh-aw/pull/55171)): `safe-outputs.create-pull-request.pre-create.steer: true` introduced run-scoped feedback issues and injected prompting for agents to read comments containing the `steer` keyword. The configuration was subsequently moved to [`safe-outputs.steer`](https://github.com/github/gh-aw/pull/55792), where it requires explicit `issues: read` without silently expanding workflow permissions.
- **`gh aw models`** ([#55148](https://github.com/github/gh-aw/pull/55148)): a new CLI command surfaces catalog pricing, alias resolution, and observed automation models in one place.
- **Copilot SDK startup diagnostics** ([#55149](https://github.com/github/gh-aw/pull/55149)): pre-ready crashes now surface the Copilot SDK's startup stderr, making a previously opaque failure mode much easier to debug.
- **Automatic PR review dismissal ingestion** ([#55180](https://github.com/github/gh-aw/pull/55180)): workflows can now ingest automatic pull request review dismissals as part of their safe-output processing.

## Notable Pull Requests

- [Align daily workflow and merged-PR metrics](https://github.com/github/gh-aw/pull/55214): standardized the daily reports' workflow population and merge-window comparisons so fleet-size and Copilot success-rate numbers stop drifting apart.
- [Fix lockfile-stats discussion category extraction and add loud self-check](https://github.com/github/gh-aw/pull/55209): the Lockfile Statistics report was silently reporting zero discussion categories due to a key mismatch in compiled `.lock.yml` parsing — now fixed, with a self-check to keep it from regressing quietly again.
- [Update detection analysis report to reflect gh-aw-detection default-on](https://github.com/github/gh-aw/pull/55236): now that `gh-aw-detection` defaults to enabled, the detection-analysis-report workflow correctly classifies unset/absent values instead of flagging them as misconfigured.
- [Migrate 30 copilot workflows to codex engine + copilot/mai-code-1-flash-picker](https://github.com/github/gh-aw/pull/55154): a large batch migration moving many Copilot-powered workflows onto the codex engine.
- [Render agentic engine in generated footers](https://github.com/github/gh-aw/pull/55192): workflow-generated content (like PR footers) now shows which engine produced it, improving traceability across the fleet.

## 🤖 Agent of the Week: PR Sous Chef

The tireless kitchen staff of the PR pipeline — it watches over open pull requests and keeps their descriptions, context, and footers fresh and useful.

This week `pr-sous-chef` ran three times in a single day (all successful, all on the `pi` engine), burning through roughly 76K tokens and racking up 8 safe-output items across its runs — including landing the new pre-create PR steering feature itself via [#55171](https://github.com/github/gh-aw/pull/55171). Its most recent run alone produced 5 safe items in under 8 minutes, a tidy little burst of productivity right before this post went out.

Somewhat fittingly, the workflow that teaches other PRs how to listen to reviewer feedback (`steer`) shipped that very feature about itself — a small bit of "eating your own dog food" that we appreciated.

💡 **Usage tip**: Reach for a `pr-sous-chef`-style workflow whenever your team's biggest bottleneck is PR descriptions and footers going stale between review rounds — it keeps that metadata current without anyone lifting a finger.

→ [View the workflow on GitHub](https://github.com/github/gh-aw/blob/main/.github/workflows/pr-sous-chef.md)

## Try It Out

Grab the latest [v0.87.4](https://github.com/github/gh-aw/releases/tag/v0.87.4) release to try the initial pre-create PR steering configuration. Current workflows use `safe-outputs.steer` as described in [#55792](https://github.com/github/gh-aw/pull/55792). As always, questions, bug reports, and contributions are welcome over at [github/gh-aw](https://github.com/github/gh-aw).
