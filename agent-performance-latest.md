# Agent Performance — Latest Run

**Timestamp:** 2026-06-23T13:40:00Z | **Run:** [§28029988746](https://github.com/github/gh-aw/actions/runs/28029988746)

## Summary: 60/100 Quality (↑ +3) | 60/100 Effectiveness (↑ +4) | 70/100 Health (↑ +4) | 3 workflows recovered

## Top Performers
1. Static Analysis Suite (Q:95, E:92) — 250 workflows scanned, dedup clean, 100% success
2. Team Status (Q:92, E:85) — daily summaries well-structured, high-value
3. Copilot SWE Agent (Q:88, E:90) — 89% merge rate (33/37 settled) ↑ from 60%
4. Agentic Maintenance (Q:85, E:88) — 100% success, reliable ops
5. Issue Monster (Q:82, E:85) — 100% success, consistent
6. Auto-Triage Issues (Q:80, E:85) — CONFIRMED RECOVERED (5+ consecutive successes)
7. PR Sous Chef (Q:80, E:83) — CONFIRMED RECOVERED (5+ successes Jun 23)
8. Bot Detection (Q:80, E:85) — 100% success

## Confirmed Recovered (since last report Jun 21)
- Auto-Triage Issues: CONFIRMED RECOVERED ✅
- PR Sous Chef: CONFIRMED RECOVERED ✅
- Daily Safe Outputs Git Simulator: CONFIRMED RECOVERED ✅

## Underperformers (Persistent)
- Code Simplifier (Q:20, E:10) P1: intermittent regression after PR #40578. #40969 filed Jun 23. DO NOT RE-FILE.
- Tool Denial Cluster (7+ workflows, Q:20, E:15): systemic. DO NOT RE-FILE.
- Daily Safe Output Integrator (Q:20, E:15) Day 15+: tool denial. #39477. DO NOT RE-FILE.
- Daily BYOK Ollama Test (Q:30, E:20) Day 15+: api-proxy cap. #39476/#40417. DO NOT RE-FILE.
- Daily News (Q:30, E:20): orphan branch signing. #40190. DO NOT RE-FILE.
- LintMonster (Q:70, E:60) Jun 23: copilot agent assignment permission failure. #40936. DO NOT RE-FILE.

## PR Merge Rates (Jun 23 window — 60 PRs)
- app/copilot-swe-agent: 89% (33/37 settled) ↑ from 60%
- app/github-actions: 91% (10/11 settled) ↑ from 50%
- app/dependabot: 0% (5 open — normal, awaiting review)

## Ecosystem Health (Jun 23)
- Workflows compiled: 250/250 (100%) ✅
- Today: 40 success, 1 intentional failure, 10 in-progress
- Smoke cluster: 36+ skipped/action_required/failing — systemic infra issue (#38998/#38999)
- Coverage: 65 unique active workflows in 100 runs

## Issues Filed This Run
- 0 new issues (all P1/P2 already tracked; discussion created)

## Coverage Gaps (unchanged)
- No automated recovery detection
- No PR stall detection (PRs open >7d without review)
- No AIC budget forecasting/alerting upstream

## Do Not Re-File (additions Jun 23)
- #40936 — LintMonster copilot agent assignment failure (monitor)
- #40969 — Code Simplifier regression (#aw_cs_regress per WH)
