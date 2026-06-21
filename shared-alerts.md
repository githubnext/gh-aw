# Shared Alerts — 2026-06-21T13:22Z (updated by Agent Performance Analyzer)

## P1 🚨
- **Tool Denial Cluster — SYSTEMIC**: 7+ workflows. Filed Jun 16. DO NOT RE-FILE.
- **Daily Model Inventory Checker — Day 11+** (#39471): session.idle 60s. DO NOT RE-FILE.
- **AIC Budget Crisis — Day 15, Code Simplifier** (#39077, #39968, #40431, #40577): api-proxy cap + HTTP 429. Fix PR #40578 open (Copilot AI, pelikhan review pending). DO NOT RE-FILE.
- **upload_artifact malformed 400** (#38998): Smoke Copilot ~75-95% failure. DO NOT RE-FILE.
- **Daily Safe Output Integrator — Day 12+** (#39477): tool denial + ECONNREFUSED. DO NOT RE-FILE.
- **Daily BYOK Ollama Test — Day 12+** (#39476, #40417): api-proxy cap. DO NOT RE-FILE.
- **Daily Cache Strategy Analyzer** (#39451): Codex model 404. ~50% alternating. DO NOT RE-FILE.
- **Smoke cluster — systemic**: 5+ engines failing. All issues filed. DO NOT RE-FILE.
- **Daily News — push_repo_memory orphan branch (#40190)**: GH013 unsigned commit. DO NOT RE-FILE.
- **Daily Compiler Quality Check (#39724/#39949)**: gpt-5-mini model unsupported. Day 4+. DO NOT RE-FILE.

## P2 ⚠️
- **aw-failures duplicate rate**: ~30-40% of failure issues are duplicates. Improvement needed.
- **LintMonster** (#39511): Alternating success/fail pattern continues.
- **PR Sous Chef** (#40586/#40548): 1 failure Jun 21. Likely transient. Watch Jun 22.
- **Content Moderation**: 25% success Jun 20 (1/4) — monitor Jun 21.
- **Agentic Commands**: 80% action_required in latest 10 runs. Watch.
- **Auto-Triage Issues** (#40598): produced no safe outputs Jun 21 (was 100%). Watch Jun 22 — possible tool denial spreading.
- **Smoke Codex** (#40600): set_issue_field cannot bind temporary_id — possible safe-outputs bug.

## Resolved ✅
- **Safe outputs target:triggering bug** (#40017): FIXED (#40035 merged Jun 18).
- **Daily Safe Outputs Git Simulator**: ✅ RECOVERED Jun 21 — 1/1 success. Hold 3 days to confirm stable.
- **Avenger**: RECOVERED Jun 20 (100%, 4/4). Monitor Jun 21.
- **Daily Documentation Updater** (#39775): HOLDING Jun 19-21.
- **Daily Workflow Updater** (#39753): HOLDING Jun 19-21.
- **Instructions Janitor** (#39757): HOLDING Jun 19-21.
- **Glossary Maintainer** (#39769): HOLDING Jun 19-21.

## Coverage Gaps (from Agent Performance Analyzer)
- No automated recovery detection
- No PR stall detection (PRs open >7d without review)
- No AIC budget forecasting/alerting upstream
- No auto-deduplication in aw-failures workflow

## Do Not Re-File (Full List)
Tool denial cluster, Code Simplifier #39199/#39489/#39729/#39968/#40431/#40577, Daily Model Inventory #39471, upload_artifact #38998, Smoke Trigger #38999, Git Simulator #39024, Failure Investigator #39037, Perf regression #38870-38872, AIC Budget #39077, Smoke Gemini #39172, Dictation #39196/#39200, Remote MCP Auth #39193/#39505, BYOK Ollama #39476/#40417, Safe Output Integrator #39477, AI Moderator #39452, Cache Strategy #39451, Compiler Threat #39343, Smoke cluster all issues, LintMonster #39511, Incomplete Result cluster, Test Quality Sentinel #39782, Matt Pocock #39781, Design Decision Gate #39779/#39776, Daily News #39758/#40023/#40074/#40190, Daily Compiler Quality #39724/#39949, Metrics Collector #39727, Instructions Janitor #39757, Avenger #40145, PR Sous Chef #40548/#40586, Auto-Triage #40598
