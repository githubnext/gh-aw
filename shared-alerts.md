# Shared Alerts — 2026-06-24T05:50Z (updated by Workflow Health Manager)

## P1 🚨
- **Code Simplifier — PERSISTENT (#40969 OPEN)**: NEW error pattern Jun 24: HTTP 403 auth at 172.30.0.30:10002. Fails daily except Jun 22 (~1/10 runs). Credential/proxy rotation suspected. Updated #40969.
- **Tool Denial Cluster — SYSTEMIC**: 7+ workflows. Filed Jun 16. DO NOT RE-FILE.
- **Daily Safe Output Integrator — Day 15+** (#39477): tool denial + ECONNREFUSED. Still failing. DO NOT RE-FILE.
- **Daily BYOK Ollama Test — Day 15+** (#39476, #40417): api-proxy cap. Still failing. DO NOT RE-FILE.
- **upload_artifact malformed 400** (#38998): Smoke Copilot ~75-95% failure. Still continuing. DO NOT RE-FILE.
- **Daily Cache Strategy Analyzer** (#39451): Alternating pattern (Jun 23 fail, Jun 22 success). DO NOT RE-FILE.
- **Daily Compiler Threat Spec Optimizer (#39343)**: Fails every 7 days, last Jun 22. Next ~Jun 29. DO NOT RE-FILE.

## P2 ⚠️ (Monitor)
- **AI Moderator "no safe outputs" (#41156)**: Single event Jun 24, run 28073081040. Previous issue #39452 CLOSED. Monitor Jun 25 for recurrence before filing.
- **Smoke cluster noise**: 36+ non-success runs creating issue tracker noise. Consider unified smoke dashboard.

## Confirmed Resolved ✅ (updated Jun 24)
- **LintMonster (#40936)**: CONFIRMED RECOVERED — Jun 24 success. ✅
- **Auto-Triage Issues** (#40598): CONFIRMED STABLE — continuing successes.
- **PR Sous Chef** (#40548/#40586): CONFIRMED STABLE — Jun 24 success.
- **Daily Safe Outputs Git Simulator**: CONFIRMED STABLE — Jun 24 success.
- **Avenger**: STABLE — continuing success.
- **Daily News** (#40190): RECOVERED — Jun 23 success.

## Agent Performance Scores (Jun 23 → Jun 24)
- Quality: 60/100 (→ stable)
- Effectiveness: 60/100 (→ stable)
- Health: 70/100 (→ stable)
- Copilot SWE merge rate: 89%
- github-actions merge rate: 91%

## Coverage Gaps (from Agent Performance Analyzer)
- No automated recovery detection
- No PR stall detection (PRs open >7d without review)
- No AIC budget forecasting/alerting upstream
- No auto-deduplication in aw-failures workflow
- No unified smoke test dashboard

## Do Not Re-File (Full List — Jun 24)
Tool denial cluster, Code Simplifier #39968/#40431/#40577 (CLOSED) + #40969 (OPEN — do not duplicate), Daily Model Inventory #39471 (RECOVERED), upload_artifact #38998, Smoke Trigger #38999, Git Simulator #39024, Failure Investigator #39037, Perf regression #38870-38872, AIC Budget #39077, Smoke Gemini #39172, Dictation #39196/#39200, Remote MCP Auth #39193/#39505, BYOK Ollama #39476/#40417, Safe Output Integrator #39477, AI Moderator #39452 (CLOSED; #41156 auto-filed Jun 24 — monitor), Cache Strategy #39451, Compiler Threat #39343, Smoke cluster all issues, LintMonster #39511/#40936 (RECOVERED Jun 24), Incomplete Result cluster, Test Quality Sentinel #39782, Matt Pocock #39781, Design Decision Gate #39779/#39776, Daily News #39758/#40023/#40074/#40190 (RECOVERED), Daily Compiler Quality #39724/#39949 (RECOVERED), Metrics Collector #39727, Instructions Janitor #39757, Avenger #40145, PR Sous Chef #40548/#40586 (RECOVERED), Auto-Triage #40598 (RECOVERED), Smoke Codex #40600
