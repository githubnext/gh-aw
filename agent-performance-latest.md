# Agent Performance — Latest Run

**Timestamp:** 2026-06-25T13:27Z | **Run:** [§28173303846](https://github.com/github/gh-aw/actions/runs/28173303846)

## Summary: 60/100 Quality (→ stable) | 60/100 Effectiveness (→ stable) | 87/100 Health (→ stable)

## Top Performers
1. Static Analysis Suite (Q:95, E:92) — 251 workflows compiled/scanned, 100% lock file coverage ✅
2. Team Status (Q:92, E:85) — well-structured, 15 commits summarized, release tracking, tables+emoji
3. Copilot SWE Agent (Q:88, E:90) — 76% merge rate (19/25 settled), 4 PRs open <1d, precise technical descriptions
4. Token Audit (Q:85, E:88) — comprehensive 30d AIC analysis (6,906 AIC, 57 active workflows)
5. Agentic Maintenance (Q:85, E:88) — 100% success today
6. Issue Monster (Q:82, E:85) — 100% success today
7. Avenger (Q:82, E:85) — confirmed stable ✅
8. PR Sous Chef (Q:80, E:83) — confirmed stable ✅
9. PR Triage Agent (Q:82, E:78) — structured output, accurate metrics, fork-only policy limits scope
10. Auto-Triage Issues (Q:80, E:75) — REGRESSION: failed today (#41450), was STABLE previously

## Persistent Underperformers
- Code Simplifier (Q:20, E:10) P1: Error type SHIFTED Jun 25 (tool denial). 5th fail in 5d. #41365 OPEN. DO NOT RE-FILE.
- Tool Denial Cluster (7+ workflows, Q:20, E:15): systemic. DO NOT RE-FILE.
- Daily Safe Output Integrator (Q:20, E:15) Day 16+: #39477. DO NOT RE-FILE.
- Daily BYOK Ollama Test (Q:30, E:20) Day 16+: #39476/#40417. DO NOT RE-FILE.
- Smoke CI upload_artifact (Q:30, E:30): #38998. DO NOT RE-FILE.

## New/Regression (Jun 25)
- Auto-Triage Issues: REGRESSION (#41450 auto-filed). Was STABLE on Jun 24. Monitor Jun 26.
- Smoke Codex/Pi/Antigravity/Copilot: Multiple engine failures (missing tools/data). Auto-filed individually.

## PR Merge Rates (Jun 25 window)
- Copilot SWE: 76% (19/25 settled closed PRs), 4 open ≤1d
- github-actions[bot]: 100% (5/5 merged)

## AIC Efficiency (30d, from token audit Jun 25)
- Total AIC: 6,906 (↓21% from Jun 24, ↓24% below 7d avg)
- Top consumer (avg): Daily AgentRx Trace Optimizer 421/run (single run)
- Top consumer (8 runs): PR Code Quality Reviewer 85/run
- Contribution Check: 213/run (2 runs) — high, monitor

## Trends
- Quality: 60/100 (→ stable)
- Effectiveness: 60/100 (→ stable)
- Health: 87/100 (→ stable)
- Compiled: 251/251 (100%) ✅
- Copilot merge rate: 76% (↓ from 89% Jun 24, different window)

## Issues Filed This Run
- 0 new issues (all known issues already tracked)

## Coverage Gaps (unchanged)
- No automated recovery detection
- No PR stall detection (PRs open >7d)
- No AIC budget forecasting/alerting upstream
- PR Triage fork-only policy renders agent largely ceremonial (all Copilot PRs are same-repo)

## Do Not Re-File
All from Jun 24 list + Code Simplifier #41365 (open) + Auto-Triage #41450 (auto-filed Jun 25)
