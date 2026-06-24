# Agent Performance — Latest Run

**Timestamp:** 2026-06-24T13:28:59Z | **Run:** [§28101831130](https://github.com/github/gh-aw/actions/runs/28101831130)

## Summary: 60/100 Quality (→ stable) | 60/100 Effectiveness (→ stable) | 72/100 Health (↑ +2)

## Top Performers
1. Static Analysis Suite (Q:95, E:92) — 251 workflows scanned, dedup clean, 100% success
2. Team Status (Q:92, E:85) — daily summaries well-structured, high-value
3. Copilot SWE Agent (Q:88, E:90) — 89% merge rate, 12 open PRs today (all <1d)
4. Agentic Maintenance (Q:85, E:88) — 100% success, reliable ops
5. Issue Monster (Q:82, E:85) — 4/4 success, consistent
6. Avenger (Q:82, E:85) — 4/4 success, confirmed stable
7. PR Sous Chef (Q:80, E:83) — 5/5 success streak, confirmed stable
8. Auto-Triage Issues (Q:80, E:85) — STABLE (5+ consecutive successes)

## Confirmed Recovered (since Jun 23)
- LintMonster: CONFIRMED RECOVERED ✅ (#40936 resolved Jun 24)
- Daily News: STABLE ✅
- Daily Safe Outputs Git Simulator: STABLE ✅

## Underperformers (Persistent)
- Code Simplifier (Q:20, E:10) P1: HTTP 403 auth at 172.30.0.30:10002 Jun 24. #40969 OPEN. DO NOT RE-FILE.
- Tool Denial Cluster (7+ workflows, Q:20, E:15): systemic. DO NOT RE-FILE.
- Daily Safe Output Integrator (Q:20, E:15) Day 15+: tool denial. #39477. DO NOT RE-FILE.
- Daily BYOK Ollama Test (Q:30, E:20) Day 15+: api-proxy cap. #39476/#40417. DO NOT RE-FILE.
- Smoke CI (Q:30, E:30): upload_artifact 400 #38998. DO NOT RE-FILE.

## Monitoring (P2)
- AI Moderator (#41156): Single "no safe outputs" Jun 24. Monitor Jun 25 before escalating.
- Daily Rendering Scripts Verifier: Failed Jun 24 (#41202 auto-filed). Monitor.
- Daily Sub-Agent Model Resolution Audit: Failed Jun 24 (#41177 auto-filed). Monitor.
- GitHub Remote MCP Auth Test (#41174): Persistent. Monitor.

## PR Merge Rates (Jun 24 window)
- app/copilot-swe-agent: 89% (33/37 settled) — stable
- app/github-actions: 91% (10/11 settled) — stable
- app/dependabot: 4 open (normal, awaiting review)
- Stale PRs (>7d): 0 ✅

## Trends
- Quality: 60/100 (→ stable)
- Effectiveness: 60/100 (→ stable)
- Health: 72/100 (↑ from 70)
- Compiled: 251/251 (100%) ✅

## Issues Filed This Run
- 0 new issues (all P1/P2 already tracked)

## Coverage Gaps (unchanged)
- No automated recovery detection
- No PR stall detection (PRs open >7d without review)
- No AIC budget forecasting/alerting upstream
- No unified smoke test dashboard (#38998 follow-up)

## Do Not Re-File (Jun 24 additions)
- LintMonster #40936 (RECOVERED Jun 24)
- All others: see shared-alerts.md
