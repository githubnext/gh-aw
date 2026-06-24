# Shared Alerts — 2026-06-24T13:28Z (updated by Agent Performance Analyzer)

## P1 🚨
- **Code Simplifier — PERSISTENT (#40969 OPEN)**: NEW error Jun 24: HTTP 403 auth at 172.30.0.30:10002 (COPILOT_PROVIDER_API_KEY rejection). Proxy credential rotation suspected. Updated #40969. DO NOT RE-FILE.
- **Tool Denial Cluster — SYSTEMIC**: 7+ workflows. Filed Jun 16. DO NOT RE-FILE.
- **Daily Safe Output Integrator — Day 15+** (#39477): tool denial + ECONNREFUSED. Still failing. DO NOT RE-FILE.
- **Daily BYOK Ollama Test — Day 15+** (#39476, #40417): api-proxy cap. Still failing. DO NOT RE-FILE.
- **upload_artifact malformed 400** (#38998): Smoke Copilot ~75-95% failure. Still continuing. DO NOT RE-FILE.
- **Daily Cache Strategy Analyzer** (#39451): Alternating pattern. DO NOT RE-FILE.
- **Daily Compiler Threat Spec Optimizer (#39343)**: Fails every 7 days. Next ~Jun 29. DO NOT RE-FILE.

## P2 ⚠️ (Monitor)
- **AI Moderator "no safe outputs" (#41156)**: Single event Jun 24, run 28073081040. Monitor Jun 25 for recurrence before any escalation.
- **Daily Rendering Scripts Verifier (#41202)**: Failed Jun 24. Auto-filed. Monitor Jun 25.
- **Daily Sub-Agent Model Resolution Audit (#41177)**: Failed Jun 24. Auto-filed. #41184 copilot PR open.
- **GitHub Remote MCP Auth Test (#41174)**: Persistent. Monitor.
- **Smoke cluster noise**: 36+ non-success runs creating issue tracker noise. Consider unified smoke dashboard (#38998 follow-up).

## Confirmed Resolved ✅ (updated Jun 24)
- **LintMonster (#40936)**: CONFIRMED RECOVERED — Jun 24 success ✅
- **Auto-Triage Issues** (#40598): CONFIRMED STABLE — continuing successes ✅
- **PR Sous Chef** (#40548/#40586): CONFIRMED STABLE — Jun 24 success ✅ (5/5 streak)
- **Daily Safe Outputs Git Simulator**: CONFIRMED STABLE — Jun 24 success ✅
- **Avenger**: STABLE — continuing success ✅
- **Daily News** (#40190): STABLE — Jun 23/24 successes ✅

## Agent Performance Scores (Jun 23 → Jun 24)
- Quality: 60/100 (→ stable)
- Effectiveness: 60/100 (→ stable)
- Health: 72/100 (↑ +2 — LintMonster recovery)
- Copilot SWE merge rate: 89%
- github-actions merge rate: 91%
- Stale PRs (>7d open): 0 ✅

## Coverage Gaps (from Agent Performance Analyzer)
- No automated recovery detection
- No PR stall detection (PRs open >7d without review)
- No AIC budget forecasting/alerting upstream
- No auto-deduplication in aw-failures workflow
- No unified smoke test dashboard

## Do Not Re-File (Full List — Jun 24)
Tool denial cluster, Code Simplifier #39968/#40431/#40577 (CLOSED) + #40969 (OPEN — do not duplicate), Daily Model Inventory #39471 (RECOVERED), upload_artifact #38998, Smoke Trigger #38999, Git Simulator #39024, Failure Investigator #39037, Perf regression #38870-38872, AIC Budget #39077, Smoke Gemini #39172, Dictation #39196/#39200, Remote MCP Auth #39193/#39505, BYOK Ollama #39476/#40417, Safe Output Integrator #39477, AI Moderator #39452 (CLOSED; #41156 auto-filed Jun 24 — monitor), Cache Strategy #39451, Compiler Threat #39343, Smoke cluster all issues, LintMonster #39511/#40936 (RECOVERED Jun 24), Incomplete Result cluster, Test Quality Sentinel #39782, Matt Pocock #39781, Design Decision Gate #39779/#39776, Daily News #39758/#40023/#40074/#40190 (RECOVERED), Daily Compiler Quality #39724/#39949 (RECOVERED), Metrics Collector #39727, Instructions Janitor #39757, Avenger #40145, PR Sous Chef #40548/#40586 (RECOVERED), Auto-Triage #40598 (RECOVERED), Smoke Codex #40600, Rendering Scripts #41202 (auto-filed Jun 24 — monitor), Sub-Agent Model Audit #41177 (auto-filed Jun 24 — monitor)
