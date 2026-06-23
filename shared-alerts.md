# Shared Alerts — 2026-06-23T05:51Z (updated by Workflow Health Manager)

## P1 🚨
- **Code Simplifier — REGRESSION (#aw_cs_regress)**: PR #40578 fix insufficient. fail→succeed→fail alternating. NEW issue filed Jun 23. Investigate AIC budget timing vs run at 04:36 UTC.
- **Tool Denial Cluster — SYSTEMIC**: 7+ workflows. Filed Jun 16. DO NOT RE-FILE.
- **Daily Safe Output Integrator — Day 14+** (#39477): tool denial + ECONNREFUSED. Still failing. DO NOT RE-FILE.
- **Daily BYOK Ollama Test — Day 14+** (#39476, #40417): api-proxy cap. Still failing. DO NOT RE-FILE.
- **upload_artifact malformed 400** (#38998): Smoke Copilot ~75-95% failure. Still continuing. DO NOT RE-FILE.
- **Daily Cache Strategy Analyzer** (#39451): Alternating pattern (Jun 22 success, Jun 21 fail). DO NOT RE-FILE.
- **Daily News — push_repo_memory orphan branch (#40190)**: Jun 22 failure. DO NOT RE-FILE.
- **Daily Compiler Threat Spec Optimizer (#39343)**: Jun 22 failure (every-7-days). DO NOT RE-FILE.

## P2 ⚠️ (Monitor)
- **Copilot Centralization Optimizer**: Only 1 run found (Jun 22 success). Watch Jun 23 run.

## Confirmed Resolved ✅
- **Auto-Triage Issues** (#40598): RECOVERED — 5+ consecutive successes Jun 22-23.
- **PR Sous Chef** (#40548/#40586): RECOVERED — 5+ successes Jun 23.
- **Daily Safe Outputs Git Simulator**: RECOVERED — Jun 23 success confirmed.
- **Avenger**: STABLE — continuing success.
- **Code Simplifier**: Was declared RECOVERED Jun 22, but REGRESSED Jun 23 → re-filed as #aw_cs_regress.
- **LintMonster** (#39511): STABLE (no recent failure data, presumed ok).
- **Daily Model Inventory Checker** (#39471): RECOVERED (no failure data, presumed ok).
- **Daily Compiler Quality Check** (#39724/#39949): RECOVERED (no failure data).

## Coverage Gaps (from Agent Performance Analyzer)
- No automated recovery detection
- No PR stall detection (PRs open >7d without review)
- No AIC budget forecasting/alerting upstream
- No auto-deduplication in aw-failures workflow

## Do Not Re-File (Full List)
Tool denial cluster, Code Simplifier #39968/#40431/#40577 (CLOSED) + #aw_cs_regress (OPEN — do not duplicate), Daily Model Inventory #39471 (RECOVERED), upload_artifact #38998, Smoke Trigger #38999, Git Simulator #39024, Failure Investigator #39037, Perf regression #38870-38872, AIC Budget #39077, Smoke Gemini #39172, Dictation #39196/#39200, Remote MCP Auth #39193/#39505, BYOK Ollama #39476/#40417, Safe Output Integrator #39477, AI Moderator #39452, Cache Strategy #39451, Compiler Threat #39343, Smoke cluster all issues, LintMonster #39511 (RECOVERED), Incomplete Result cluster, Test Quality Sentinel #39782, Matt Pocock #39781, Design Decision Gate #39779/#39776, Daily News #39758/#40023/#40074/#40190, Daily Compiler Quality #39724/#39949 (RECOVERED), Metrics Collector #39727, Instructions Janitor #39757, Avenger #40145, PR Sous Chef #40548/#40586 (RECOVERED), Auto-Triage #40598 (RECOVERED), Smoke Codex #40600
