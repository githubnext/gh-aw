# Shared Alerts — 2026-06-29T06:05Z (updated by Workflow Health Manager)

## P1 🚨
- **Sub-Agent Model Resolution Audit (#42033 OPEN)**: 100% red since Jun 24. Codex `gpt-5-codex-alpha-2025-11-07` returns 404 — retired alpha snapshot. Repo-wide fix needed: stop pinning alpha model snapshots. DO NOT RE-FILE.
- **PR Code Quality Reviewer (#42095 OPEN)**: Tier-unsupported model via `general-purpose` subagent → SDK 400. Sub-issue of #42033. DO NOT RE-FILE.
- **Daily Safe Output Integrator (#42125 OPEN)**: Tool denial 5/5 AGAIN — 3rd recurring instance (prev: #41935, #41827-related). Structural refactor needed. DO NOT RE-FILE.
- **Daily BYOK Ollama Test (#41827 OPEN)**: api-proxy 503 on /v1/models. 9+ days infra outage. DO NOT RE-FILE.
- **Go Logger Enhancement (#42032 OPEN)**: Pre-agent `jq` step: ARG_MAX exceeded, agent never starts. DO NOT RE-FILE.

## P2 ⚠️ (Monitor)
- **Changeset Generator (#41987 OPEN)**: Push rejected — `workflows` scope needed.
- **Smoke Copilot (#41988 OPEN)**: dispatch_workflow missing `message` input. 1 failure Jun 27 22:13, monitor.
- **Daily Formal Spec Verifier (#42105 OPEN)**: Tool denial 5/5 — same pattern as Safe Output Integrator.
- **Agentic Workflow Audit Agent (#42140 OPEN)**: Invalid/unsupported model in config — config fix needed.
- **Daily Code Metrics (#42124 OPEN)**: Missing `upload_asset` tool.
- **Daily Team Evolution Insights (#42128 OPEN)**: Missing GitHub MCP read tools.

## Confirmed Stable ✅
- **CI**: STABLE ✅ — passing Jun 29 06:03. Issue #41844 (nolint-suppression gap, different topic) remains open.
- **Compilation**: 257/257 ✅ STABLE
- **Avenger**: STABLE ✅ (recovered after Jun 28 intermittency — success Jun 29 06:03)
- **Auto-Triage Issues**: STABLE ✅
- **PR Sous Chef**: STABLE ✅
- **Copilot SWE Agent**: HIGH-PERFORMING ✅ (80% merge rate)

## Health Scores (Jun 29 06:05Z)
- Compilation: 257/257 ✅
- Health Score: 80/100 (↓2)
- P1 issues: 5 (Sub-Agent Model, PR Code Quality, Safe Output Integrator, BYOK Ollama, Go Logger)
- P2 issues: 6 (Changeset Generator, Smoke Copilot, Formal Spec Verifier, Audit Workflows, Code Metrics, Team Evolution)

## Systemic Issues
1. **Codex alpha model 404**: `gpt-5-codex-alpha-2025-11-07` decommissioned — affects Sub-Agent Model Audit, Cache Strategy Analyzer, PR Code Quality; tracked in #42033
2. **Tool denial guardrails 5/5**: Structural issue — Safe Output Integrator, Formal Spec Verifier need complexity reduction
3. **Missing tool declarations**: Code Metrics, Team Evolution — audit daily workflow frontmatter
4. **Tier-unsupported model**: Subagent model resolution must validate against supported-model set (#42095)

## Recoveries Jun 28–29
- CI: STABLE ✅ (PR #41849 holding)
- Avenger: RECOVERED ✅ (was intermittent Jun 28 09:30-11:58, stable Jun 29)
- Code Simplifier: Issues closed Jun 28 — monitor for new instances

## Update — 2026-06-29T14:01Z (Agent Performance Analyzer)
- **Escalating tool denial pattern**: 3 agents now: Safe Output Integrator (#42125), Formal Spec Verifier (#42105), Layout Spec Maintainer (#42204). Systemic issue filed: #aw_toolden1. 
- **AI Moderator**: #42234 filed, 0/4 success, 2 ar, 2 skip.
- **Q workflow**: 10/14 action_required — monitor; appears by-design (dispatch approval flow).
- Copilot SWE Agent STABLE: 80% merge rate maintained.
