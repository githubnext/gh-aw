# Shared Alerts — 2026-06-28T05:54Z (updated by Workflow Health Manager)

## P1 🚨
- **Code Simplifier (#42003 OPEN)**: 8+ consecutive failures (Jun 20–Jun 28). FIX PR #41852 MERGED but did NOT resolve (run Jun 28 04:45 still EACCES rimraf). Deeper investigation needed. DO NOT RE-FILE.
- **Daily Safe Output Integrator (#41935 OPEN)**: 6+ consecutive failures (Jun 23–27). Tool denial limit exceeded (5/5). Needs prompt/config refactor. DO NOT RE-FILE.
- **Daily BYOK Ollama Test (#41827 OPEN)**: 9+ consecutive failures. api-proxy returns 503 on /v1/models. Infra dependency. DO NOT RE-FILE.

## P2 ⚠️ (Monitor)
- **Go Logger Enhancement (#42002 OPEN)**: 3 consecutive failures (Jun 26–28). WIP stabilization.
- **Smoke Copilot (#41988 OPEN)**: 1 failure Jun 27 22:13 (dispatch_workflow missing `message` input), recovered 22:59. Monitor.
- **Changeset Generator (#41987 OPEN)**: Push rejected — needs `workflows` scope on token. Monitor.

## Confirmed Stable ✅
- **CI**: RESOLVED ✅ — PR #41849 merged, passing since Jun 28 03:17 (issue #41844 can be closed)
- **Auto-Triage Issues**: STABLE ✅
- **Compilation**: 257/257 ✅ STABLE
- **PR Sous Chef**: STABLE ✅
- **Avenger**: STABLE ✅

## Health Scores (Jun 28 05:54Z)
- Compilation: 257/257 ✅
- Health Score: 82/100 (↓2 from 84)
- P1 issues: 3 (Code Simplifier, BYOK Ollama, Safe Output Integrator)
- P2 issues: 3 monitored (Go Logger, Smoke Copilot, Changeset Generator)
- Recoveries: CI ✅ RESOLVED

## Systemic Issues (For Campaign Manager)
1. **Chroot EACCES** (Code Simplifier): Fix PR #41852 did not take effect — may affect other workflows using chroot. High priority.
2. **Tool denial guardrails**: Safe Output Integrator structural refactor needed
3. **Safe-output push scope**: Changeset Generator needs `workflows` scope — may affect other workflows pushing lock files
