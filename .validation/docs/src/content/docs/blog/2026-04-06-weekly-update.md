---
title: "Weekly Update – April 6, 2026"
description: "Ten releases in seven days: full OpenTelemetry distributed tracing, a new report_incomplete safe output, Claude Code 1.0.0 support, and security hardening across the board."
authors:
  - copilot
date: 2026-04-06
---

Ten releases landed in [github/gh-aw](https://github.com/github/gh-aw) between March 31 and April 6 — a relentless pace that delivered production-ready distributed tracing, new safe output signals, and a sweeping security cleanup. Here's what shipped.

## Release Highlights

### [v0.67.1](https://github.com/github/gh-aw/releases/tag/v0.67.1) — OpenTelemetry Overhaul & Security Hardening (April 6)

The headline release of the week polishes the OTLP tracing story introduced in v0.67.0 and adds a wave of security fixes.

- **Accurate span names and real job durations** ([#24823](https://github.com/github/gh-aw/pull/24823)): Job lifecycle spans now use the actual job name (e.g. `gh-aw.agent.conclusion`) and record real execution time — previously spans always reported 2–5 ms due to a missing `startMs`.
- **OTLP payload sanitization**: Sensitive values (`token`, `secret`, `key`, `auth`, etc.) in span attributes are automatically redacted before sending to any OTLP collector.
- **OTLP headers masking** ([#24805](https://github.com/github/gh-aw/pull/24805)): `OTEL_EXPORTER_OTLP_HEADERS` is masked with `::add-mask::` in every job, preventing auth tokens from leaking into GitHub Actions debug logs.
- **MCP Gateway OpenTelemetry** ([#24697](https://github.com/github/gh-aw/pull/24697)): The MCP Gateway now receives OpenTelemetry config derived from `observability.otlp` frontmatter and the `actions/setup` trace IDs, correlating all MCP tool-call traces under the workflow root trace.
- **`report_incomplete` safe output** ([#24796](https://github.com/github/gh-aw/pull/24796)): A new first-class signal lets agents surface infrastructure or tool failures without being misclassified as successful runs. When an agent emits `report_incomplete`, the safe-outputs handler activates failure handling regardless of agent exit code.
- **`checks` as a first-class MCP tool** ([#24818](https://github.com/github/gh-aw/pull/24818)): The `checks` tool is now registered in the gh-aw MCP server, returning a normalized CI verdict (`success`, `failed`, `pending`, `no_checks`, `policy_blocked`).
- **Token/secret injection prevention**: 422 instances of `${{ secrets.* }}` interpolated directly into `run:` blocks were moved to `env:` mappings across lock files.
- **Claude Code 1.0.0 compatibility** ([#24807](https://github.com/github/gh-aw/pull/24807)): Removed the `--disable-slash-commands` flag that was dropped in Claude Code 1.0.0.

### [v0.67.0](https://github.com/github/gh-aw/releases/tag/v0.67.0) — OTLP Trace Export & GitHub API Rate Limit Analytics (April 5)

The milestone release that first shipped distributed tracing support:

- **`observability.otlp` frontmatter**: Workflows can now export structured OpenTelemetry spans to any OTLP-compatible backend (Honeycomb, Grafana Tempo, Sentry) with a single frontmatter block. Every job emits setup and conclusion spans; cross-job trace correlation is wired automatically with a single trace ID from the activation job.
- **GitHub API rate limit analytics**: `gh aw audit`, `gh aw logs`, and `gh aw audit diff` now show GitHub API quota consumed per run, per resource.
- **Environment Variable Reference**: A new comprehensive reference section covers all CLI configuration variables.

### [v0.66.1](https://github.com/github/gh-aw/releases/tag/v0.66.1) — Richer `gh aw logs` & Breaking Change (April 4)

**⚠️ Breaking change**: `gh aw audit report` has been removed. Cross-run security reports are now generated directly by `gh aw logs --format`. The new `--last` flag aliases `--count` to ease migration.

- **Flat run classification** in `gh aw logs --json`: Each run now carries a top-level `classification` string (`"risky"`, `"normal"`, `"baseline"`, or `"unclassified"`), eliminating null-guard gymnastics.
- **Per-tool-call metrics in logs**: Granular token usage, failure counts, and latency per tool — perfect for identifying which tools consume the most resources.

### [v0.66.0](https://github.com/github/gh-aw/releases/tag/v0.66.0) — Token Usage Artifacts & Threat Detection Extensibility (April 3)

- **Token Usage Artifact** ([#24315](https://github.com/github/gh-aw/pull/24315)): Agent token usage is now uploaded as a workflow artifact, making it easy to track spend over time.
- Workflow reliability and threat detection extensibility improvements shipped alongside.

### Earlier in the week

[v0.65.7](https://github.com/github/gh-aw/releases/tag/v0.65.7) through [v0.65.2](https://github.com/github/gh-aw/releases/tag/v0.65.2) (March 31–April 3) focused on cross-repo workflow reliability, MCP gateway keepalive configuration, safe-outputs improvements, and token optimization tooling.

---

## 🤖 Agent of the Week: agentic-observability-kit

The tireless watchdog that monitors your entire fleet of agentic workflows and escalates when things go sideways.

Every day, `agentic-observability-kit` pulls logs from all running workflows, classifies their behavior, and posts a structured observability report as a GitHub Discussion — then files issues when patterns of waste or failure cross defined thresholds. This past week it had a particularly eventful run: on April 6 it spotted that `smoke-copilot` and `smoke-claude` had each burned through 675K–1.7M tokens across multiple runs (flagged as `resource_heavy_for_domain` with high severity), and it filed an issue titled *"Smoke Copilot and Smoke Claude repeatedly resource-heavy"* before anyone on the team had noticed. It also caught that the GitHub Remote MCP Authentication Test workflow had a 100% failure rate across two runs — one of which completed at zero tokens, suggesting a config or auth problem rather than an agent misbehaving.

In a delightfully meta moment, the observability kit itself hit token-limit errors while trying to ingest its own log data — it made four attempts with progressively smaller `count` and `max_tokens` parameters before it could fit the output into context. It got there in the end.

💡 **Usage tip**: Pair `agentic-observability-kit` with Slack or email notifications so escalation issues trigger an alert — otherwise the issues it files can sit unread while the token bill quietly grows.

→ [View the workflow on GitHub](https://github.com/github/gh-aw/blob/main/.github/workflows/agentic-observability-kit.md)

---

## Try It Out

Update to [v0.67.1](https://github.com/github/gh-aw/releases/tag/v0.67.1) and start exporting traces from your workflows today — all it takes is an `observability.otlp` block in your frontmatter. Feedback and contributions are always welcome in [github/gh-aw](https://github.com/github/gh-aw).
