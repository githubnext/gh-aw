# Shared Alerts — 2026-06-25T05:51Z (updated by Workflow Health Manager)

## P1 🚨
- **Code Simplifier — PERSISTENT (#41365 OPEN)**: Error type SHIFTED Jun 25: tool denial (8 denied cmds) vs Jun 24 HTTP 403 auth. 5th failure in 5 days. Issue #40969 auto-closed Jun 24 → new #41365 auto-filed Jun 25. Comment added with history. DO NOT RE-FILE.
- **Tool Denial Cluster — SYSTEMIC**: 7+ workflows. PR Description Updater hit today (6 denied cmds, auto-file failed 403). Code Simplifier hitting both auth AND tool denial. DO NOT RE-FILE separately.
- **Daily Safe Output Integrator — Day 16+** (#39477): tool denial + ECONNREFUSED. Still failing. DO NOT RE-FILE.
- **Daily BYOK Ollama Test — Day 16+** (#39476, #40417): api-proxy cap. Still failing. DO NOT RE-FILE.
- **upload_artifact malformed 400** (#38998): Smoke Copilot ~75-95% failure. Still continuing. DO NOT RE-FILE.
- **Daily Cache Strategy Analyzer** (#39451): Alternating pattern. DO NOT RE-FILE.
- **Daily Compiler Threat Spec Optimizer (#39343)**: Fails every ~7 days. Next ~Jun 29. DO NOT RE-FILE.

## P2 ⚠️ (Monitor)
- **Issue Monster "Copilot agent unavailable" (#41381)**: Auto-filed Jun 25. Workflow succeeds; assignment fails for #41256, #41061. Monitor for pattern.
- **Daily Rendering Scripts Verifier**: Issue #41202 auto-closed. Failed Jun 24 (isolated). Jun 25 run not yet (~08:40 UTC). Monitor.
- **Daily Sub-Agent Model Resolution Audit**: Issue #41177 auto-closed. Failed Jun 24 (only 1 run). Jun 25 run not yet. Monitor.
- **GitHub Remote MCP Auth Test**: Issue #41174 auto-closed. Failed Jun 24 (isolated, 1/10). Jun 25 run not yet. Monitor.
- **Daily Compiler Threat Spec Optimizer (#39343)**: Next expected failure ~Jun 29. Monitor.

## Confirmed Resolved ✅ (updated Jun 25)
- **LintMonster (#40936)**: CONFIRMED RECOVERED ✅
- **Auto-Triage Issues** (#40598): CONFIRMED STABLE ✅
- **PR Sous Chef** (#40548/#40586): CONFIRMED STABLE ✅ (6+ streak)
- **Daily Safe Outputs Git Simulator**: CONFIRMED STABLE ✅
- **Avenger**: STABLE ✅
- **Daily News** (#40190): STABLE ✅
- **AI Moderator #41156**: Auto-closed. action_required runs are normal GitHub behavior. 2 success runs Jun 25 ✅

## Agent Performance Scores (Jun 24 → Jun 25)
- Quality: 60/100 (→ stable)
- Effectiveness: 60/100 (→ stable)
- Health: 87/100 (→ stable)
- Copilot SWE merge rate: ~89% (last measured Jun 24)
- Stale PRs (>7d): 0 ✅ (last measured Jun 24)
