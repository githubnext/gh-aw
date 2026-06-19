# Shared Alerts — 2026-06-19T06:11Z (updated by Workflow Health Manager)

## P1 🚨
- **Tool Denial Cluster — SYSTEMIC**: 7+ workflows. Systemic issue filed Jun 16. DO NOT RE-FILE.
- **Daily Model Inventory Checker — Day 10** (#39471): session.idle 60s. DO NOT RE-FILE.
- **AIC Budget Crisis — Day 12, Code Simplifier** (#39077, #39199, #39489, #39729, #39968): api-proxy cap + HTTP 429. Root fix #39479 still pending. DO NOT RE-FILE.
- **upload_artifact malformed 400** (#38998): Smoke Copilot ~75-95% failure. DO NOT RE-FILE.
- **Daily Safe Outputs Git Simulator — Day 11+**: branch missing. DO NOT RE-FILE.
- **Smoke Gemini** (#39172): preview TTS model. DO NOT RE-FILE.
- **Daily BYOK Ollama Test — Day 11** (#39476): transient_bad_request. DO NOT RE-FILE.
- **Daily Safe Output Integrator — Day 11** (#39477): tool denial + ECONNREFUSED. DO NOT RE-FILE.
- **Daily Cache Strategy Analyzer** (#39451): Codex model 404. ~50% alternating. DO NOT RE-FILE.
- **Smoke cluster — systemic**: 5+ engines failing. DO NOT RE-FILE (all issues filed Jun 18).
- **Failure Detection Deduplication (#aw_dedup)**: ~30-40% of failure issues are duplicates. Systemic issue filed Jun 18.
- **Daily News — push_repo_memory orphan branch (#40190)**: GH013 unsigned commit on orphan-branch first push. Filed Jun 19 by aw-failure-investigator. DO NOT RE-FILE.

## P2 ⚠️
- **Daily Compiler Quality Check (Day 3)**: gpt-5-mini model unsupported. Easy config fix needed.
- **Daily News (Day 7+)** (#39758, #40190): orphan branch signing now primary failure mode. PR #40033 status unclear.
- **LintMonster** (#39511): Alternating success/fail + 3 new issues today.
- **Daily Cache Strategy Analyzer** (#39451): ~50% alternating.

## Resolved ✅
- **Safe outputs target:triggering bug** (#40017): FIXED (#40035 merged Jun 18).
- **Daily Documentation Updater** (#39775): FULLY RECOVERED — holding Jun 19.
- **Daily Workflow Updater** (#39753): FULLY RECOVERED — holding Jun 19.
- **Instructions Janitor** (#39757): RECOVERED — holding Jun 19.
- **Glossary Maintainer** (#39769): RECOVERED — holding Jun 19.
- **PR Sous Chef**: FULLY RECOVERED (ongoing).
- **Avenger**: Healthy.
- **AI Moderator**: Recovering.

## Do Not Re-File (Full List)
Tool denial cluster, Code Simplifier #39199/#39489/#39729/#39968, Daily Model Inventory #39471, upload_artifact #38998, Smoke Trigger #38999, Git Simulator #39024, Failure Investigator #39037, Perf regression #38870-38872, AIC Budget #39077, Smoke Gemini #39172, Dictation #39196/#39200, Remote MCP Auth #39193/#39505, BYOK Ollama #39476, Safe Output Integrator #39477, AI Moderator #39452, Cache Strategy #39451, Compiler Threat #39343, Smoke cluster all issues, LintMonster #39511, Incomplete Result cluster, Test Quality Sentinel #39782, Matt Pocock #39781, Design Decision Gate #39779/#39776, Daily News #39758/#40023/#40074/#40190, Daily Compiler Quality #39724/#39949, Metrics Collector #39727, Instructions Janitor #39757
