# Shared Alerts — 2026-06-27T13:05Z (updated by Agent Performance Analyzer)

## P1 🚨
- **Code Simplifier (#41842 OPEN, WIP PR #41852)**: 6th consecutive failure (last success Jun 22). EACCES on chroot-home cleanup. WIP fix in `copilot/aw-fix-code-simplifier-failure`. ~1.9M tokens wasted/run. DO NOT RE-FILE.
- **Daily Safe Output Integrator (#41788 OPEN)**: 6+ consecutive failures (Jun 22-27). Tool denial limit exceeded. Needs prompt/config refactor. DO NOT RE-FILE.
- **Daily BYOK Ollama Test (#41827+#41811 OPEN)**: 8+ consecutive failures. api-proxy returns 503 on /v1/models. Infra dependency. DO NOT RE-FILE.
- **CI Regression (#41844 OPEN)**: CI schedule failing Jun 27 (01:02, 03:15 UTC). Root cause: nolint-suppression parity gap. WIP PR `copilot/fix-nolint-suppression-gap` (#41849). DO NOT RE-FILE.

## P2 ⚠️ (Monitor)
- **Go Logger Enhancement (#41839 OPEN)**: 2 consecutive failures (Jun 26, 27). WIP stabilization.
- **Agentic Workflow Audit Agent (#41807 OPEN)**: 1 failure Jun 26. Closed Jun 27 — confirmed transient.
- **Daily Cache Strategy Analyzer (#41787 OPEN)**: Alternating pattern (fail/success). Flaky — needs root cause investigation.
- **Daily yamllint Fixer (#41825 OPEN)**: 1 failure Jun 27 dispatch. Likely transient, monitor.
- **Design Decision Gate (#41832 OPEN)**: PR-triggered failure Jun 27. Monitor.

## Confirmed Stable ✅
- **Auto-Triage Issues**: FULLY STABLE ✅ — 5+ consecutive successes Jun 26-27
- **Compilation**: 253/253 ✅ STABLE
- **PR Sous Chef**: STABLE ✅
- **Daily Safe Outputs Git Simulator**: STABLE ✅
- **Avenger**: STABLE ✅ (4/5 recent, last successful)
- **Issue Monster**: STABLE ✅ (5/5 streak)
- **PR Triage Agent**: STABLE ✅ (5/5)
- **Team Status**: STABLE ✅ (5/5)

## Health Scores (Jun 27 13:05Z)
- Compilation: 253/253 ✅
- Overall Quality Score: 62/100 (↑+2)
- Effectiveness Score: 63/100 (↑+3)
- Health Score: 84/100 (↓3 — CI regression)
- P1 issues: 4 (Code Simplifier, BYOK Ollama, Safe Output Integrator, CI Regression)
- P2 issues: ~5 monitored
- Recoveries: Auto-Triage fully stable ✅
- PR merge rate (swe-agent): 81% (↑ from ~74%)
- Fleet AIC/24h: 6,812 (↓1.4%)

## Recommendations (For Campaign Manager)
1. Code Simplifier WIP PR #41852 needs merge priority — blocking 1.9M AIC/run waste
2. Safe Output Integrator needs campaign to restructure tool usage (structural refactor)
3. Stale PR detection gap — no campaign coverage yet
