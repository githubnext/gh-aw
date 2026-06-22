# Shared Alerts — 2026-06-22T06:15Z (updated by Workflow Health Manager)

## P1 🚨
- **Tool Denial Cluster — SYSTEMIC**: 7+ workflows. Filed Jun 16. DO NOT RE-FILE.
- **Daily Safe Output Integrator — Day 13+** (#39477): tool denial + ECONNREFUSED. Still failing. DO NOT RE-FILE.
- **Daily BYOK Ollama Test — Day 13+** (#39476, #40417): api-proxy cap. Still failing. DO NOT RE-FILE.
- **AIC Budget Crisis — PARTIALLY RESOLVED**: Code Simplifier fix PR #40578 merged Jun 21. Code Simplifier RECOVERED Jun 22. Other api-proxy workflows may still be affected. Issue #39077.
- **upload_artifact malformed 400** (#38998): Smoke Copilot ~75-95% failure. Still continuing. DO NOT RE-FILE.
- **Daily Cache Strategy Analyzer** (#39451): Codex model 404. ~50% alternating. DO NOT RE-FILE.
- **Smoke cluster — systemic**: 5+ engines failing. All issues filed. DO NOT RE-FILE.
- **Daily News — push_repo_memory orphan branch (#40190)**: GH013 unsigned commit. Not run for 3 days. DO NOT RE-FILE.
- **Daily Compiler Threat Spec Optimizer (#39343)**: Every-7-days failure pattern. Last Jun 22. DO NOT RE-FILE.

## P2 ⚠️ (Monitor)
- **Daily Community Attribution Updater**: 1 failure Jun 22 (transient). Watch Jun 23 — do NOT file unless 2+ consecutive.
- **Daily Safe Outputs Git Simulator**: Day 2 of recovery (Jun 21-22). Watch to confirm stable 3 days.
- **PR Sous Chef** (#40586/#40548): Jun 22 = success. Appears transient.
- **Auto-Triage Issues** (#40598): Jun 22 = success (4/4). Appears recovered. Watch Jun 23.
- **Agentic Commands**: Persistent action_required pattern. Not new. DO NOT RE-FILE.
- **Copilot Centralization Optimizer**: 1 transient failure Jun 22, self-recovered. Watch.

## Resolved ✅
- **Code Simplifier**: RECOVERED Jun 22 — PR #40578 merged Jun 21 fixed root cause.
- **Daily Model Inventory Checker** (#39471): RECOVERED Jun 20-22 (3 days). Monitor 3 more days.
- **Daily Compiler Quality Check** (#39724/#39949): RECOVERED Jun 20-22 (3 days). Confirmed stable.
- **LintMonster** (#39511): STABILIZED Jun 20-22 (3 days). Confirmed stable.
- **Safe outputs target:triggering bug** (#40017): FIXED (#40035 merged Jun 18).
- **Daily Safe Outputs Git Simulator**: Day 2 recovery (hold for confirmation).
- **Avenger**: RECOVERED Jun 20+. Jun 22 = success.

## Coverage Gaps (from Agent Performance Analyzer)
- No automated recovery detection
- No PR stall detection (PRs open >7d without review)
- No AIC budget forecasting/alerting upstream
- No auto-deduplication in aw-failures workflow

## Do Not Re-File (Full List)
Tool denial cluster, Code Simplifier #39199/#39489/#39729/#39968/#40431/#40577 (RECOVERED), Daily Model Inventory #39471 (RECOVERED), upload_artifact #38998, Smoke Trigger #38999, Git Simulator #39024, Failure Investigator #39037, Perf regression #38870-38872, AIC Budget #39077, Smoke Gemini #39172, Dictation #39196/#39200, Remote MCP Auth #39193/#39505, BYOK Ollama #39476/#40417, Safe Output Integrator #39477, AI Moderator #39452, Cache Strategy #39451, Compiler Threat #39343, Smoke cluster all issues, LintMonster #39511 (RECOVERED), Incomplete Result cluster, Test Quality Sentinel #39782, Matt Pocock #39781, Design Decision Gate #39779/#39776, Daily News #39758/#40023/#40074/#40190, Daily Compiler Quality #39724/#39949 (RECOVERED), Metrics Collector #39727, Instructions Janitor #39757, Avenger #40145, PR Sous Chef #40548/#40586, Auto-Triage #40598, Smoke Codex #40600
