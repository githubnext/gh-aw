# Shared Alerts — 2026-06-25T13:27Z (updated by Agent Performance Analyzer)

## P1 🚨
- **Code Simplifier — PERSISTENT (#41365 OPEN)**: Error type SHIFTED Jun 25: tool denial (8 denied cmds) vs Jun 24 HTTP 403 auth. 5th failure in 5 days. DO NOT RE-FILE.
- **Tool Denial Cluster — SYSTEMIC**: 7+ workflows. Code Simplifier, PR Description Updater, and others. DO NOT RE-FILE separately.
- **Daily Safe Output Integrator — Day 16+** (#39477): tool denial + ECONNREFUSED. Still failing. DO NOT RE-FILE.
- **Daily BYOK Ollama Test — Day 16+** (#39476, #40417): api-proxy cap. Still failing. DO NOT RE-FILE.
- **upload_artifact malformed 400** (#38998): Smoke Copilot ~75-95% failure. Still continuing. DO NOT RE-FILE.
- **Daily Cache Strategy Analyzer** (#39451): Alternating pattern. DO NOT RE-FILE.
- **Daily Compiler Threat Spec Optimizer (#39343)**: Fails every ~7 days. Next ~Jun 29. DO NOT RE-FILE.

## P2 ⚠️ (Monitor)
- **Auto-Triage Issues REGRESSION** (#41450 auto-filed Jun 25): Was STABLE Jun 24. Single failure. Monitor Jun 26 before escalating.
- **Smoke Engines (Codex, Pi, Antigravity, Copilot AOAI Entra, Copilot)**: Multiple engine failures Jun 25 - missing tools/data. Multiple auto-filed issues. Monitor for pattern.
- **Issue Monster "Copilot agent unavailable" (#41381)**: Workflow succeeds; assignment fails for some issues. Normal operational behavior.
- **Daily Rendering Scripts Verifier**: Failed Jun 24. Monitor Jun 25 run.
- **Daily Sub-Agent Model Resolution Audit**: Failed Jun 24. Monitor Jun 25 run.
- **GitHub Remote MCP Auth Test**: Failed Jun 24. Monitor Jun 25 run.

## Confirmed Resolved ✅
- **LintMonster (#40936)**: CONFIRMED RECOVERED ✅
- **Auto-Triage Issues**: was STABLE; NEW REGRESSION Jun 25 (#41450) — moved to P2 Monitor
- **PR Sous Chef** (#40548/#40586): CONFIRMED STABLE ✅ (6+ streak)
- **Daily Safe Outputs Git Simulator**: CONFIRMED STABLE ✅
- **Avenger**: STABLE ✅
- **Daily News** (#40190): STABLE ✅
- **AI Moderator #41156**: RESOLVED. action_required runs are normal GitHub behavior. ✅

## Agent Performance Scores (Jun 25 final)
- Quality: 60/100 (→ stable)
- Effectiveness: 60/100 (→ stable)
- Health: 87/100 (→ stable)
- Copilot SWE merge rate: 76% (19/25 settled, Jun 25 window)
- AIC 30d: 6,906 (↓21% from Jun 24)
- Stale PRs (>7d): 0 ✅
