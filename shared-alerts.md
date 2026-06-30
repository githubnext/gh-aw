# Shared Alerts — 2026-06-30T05:52Z (updated by Workflow Health Manager)

## P1 🚨
- **CI Integration Test Regression (NEW, Jun 30)**: `TestMCPGatewayDockerCommandUsesRunnerIdentityAndSocketGroup` failing on main, run §28422589916. "Docker command should map host.docker.internal to host-gateway". CI passed at 01:03 and 03:15, failed 05:30. New issue filed.
- **Sub-Agent Model Resolution Audit (#42033 OPEN)**: 100% red since Jun 24. Codex `gpt-5-codex-alpha-2025-11-07` returns 404. DO NOT RE-FILE.
- **PR Code Quality Reviewer (#42095 OPEN)**: Tier-unsupported model → SDK 400. DO NOT RE-FILE.
- **Daily Safe Output Integrator (#42333 OPEN)**: Tool denial 5/5 again (4th instance). DO NOT RE-FILE.
- **Daily BYOK Ollama Test (#41827 OPEN)**: api-proxy 503. 10+ days infra outage. DO NOT RE-FILE.
- **Go Logger Enhancement (#42032 OPEN)**: jq ARG_MAX. DO NOT RE-FILE.

## P2 ⚠️ (Monitor)
- **Changeset Generator (#41987 OPEN)**: Push rejected — `workflows` scope needed. DO NOT RE-FILE.
- **Smoke Copilot (#41988 OPEN)**: dispatch_workflow missing `message` input. DO NOT RE-FILE.
- **PR Sous Chef (#42370 OPEN)**: 1 failure Jun 30 05:27. Single transient failure. DO NOT RE-FILE.
- **Agentic Workflow Audit Agent (#42356 OPEN)**: Recurring failure Jun 29. DO NOT RE-FILE.
- **Daily Team Evolution Insights (#42342 OPEN)**: Missing required tool (re-recurrence). DO NOT RE-FILE.
- **Smoke CI hard-red (#42398 OPEN)**: EACCES mkdir /tmp/gh-aw (rootless sandbox issue). DO NOT RE-FILE.
- **Copilot Opt (#42329 OPEN)**: Tool denial limit. DO NOT RE-FILE.
- **AI Moderator (#42332 OPEN)**: Incomplete result. DO NOT RE-FILE.

## Confirmed Stable ✅
- **Compilation**: 257/257 ✅ STABLE
- **Avenger**: Running Jun 30 05:49
- **Auto-Triage Issues**: STABLE ✅

## Health Scores (Jun 30 05:52Z)
- Compilation: 257/257 ✅
- Health Score: 78/100 (↓2)
- P1 issues: 6 (CI test regression NEW + Sub-Agent Model, PR Code Quality, Safe Output Integrator, BYOK Ollama, Go Logger)
- P2 issues: 8

## Systemic Issues
1. **Codex alpha model 404**: tracked #42033
2. **Tool denial guardrails 5/5**: 4th+ recurrence, escalating — Safe Output Integrator, Copilot Opt, Formal Spec Verifier (closed) → systemic refactor urgently needed
3. **Missing tool declarations**: Team Evolution #42342
4. **Smoke CI rootless sandbox EACCES**: #42398

## Recoveries Jun 29-30
- Many P2 issues closed (8 closures Jun 29)
- Layout Spec Maintainer: CLOSED #42204
- AI Moderator old: CLOSED #42234
- GitHub MCP Structural Analysis: CLOSED #42248
