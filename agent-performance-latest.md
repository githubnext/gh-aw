# Agent Performance — Latest Run

**Timestamp:** 2026-06-27T13:05Z | **Run:** [§28289921020](https://github.com/github/gh-aw/actions/runs/28289921020)

## Summary: 62/100 Quality (↑+2) | 63/100 Effectiveness (↑+3) | 84/100 Health (↓3)

## Top Performers
1. Copilot SWE Agent (Q:92, E:91) — 81% merge rate (62/77 settled), 87 PRs in window
2. PR Triage Agent (Q:83, E:85) — 5/5 runs, structured reports with delta tracking
3. Team Status (Q:85, E:82) — 5/5 runs, 15 commits summarized Jun 27
4. PR Code Quality Reviewer (Q:85, E:80) — 4/5 runs, 94.75 AIC/run deep review
5. Issue Monster / Agentic Maintenance (Q:82, E:85) — 5/5 runs, 100% success streak
6. Plan Command (Q:80, E:78) — 5/5 runs, sub-issue grouping working
7. Smoke Copilot (Q:78, E:80) — 5/5 runs, 512.6 AIC/run

## Recovery
- **Auto-Triage Issues: FULLY STABLE** ✅ (was P1 last week, now 5+ consecutive successes)
- Issue #41570 resolved/closed

## Persistent Underperformers
- Code Simplifier (Q:10, E:5) P1: 6th consecutive failure (EACCES chroot cleanup). WIP PR #41852. Issue #41842 OPEN. DO NOT RE-FILE.
- Daily Safe Output Integrator (Q:10, E:10): tool denial limit. Issue #41788 OPEN. DO NOT RE-FILE.
- Daily BYOK Ollama (Q:20, E:15): api-proxy 503. Issues #41827+#41811 OPEN. DO NOT RE-FILE.

## AIC Efficiency (Jun 27 24h)
- Total AIC: 6,812 (↓1.4% from 6,906 Jun 26) | 100 runs | 60 workflows
- Code Simplifier wasted: ~1.9M tokens/failed run
- PR Code Quality Reviewer: 94.75/run (justified)

## PR Stats (last 100)
- copilot-swe-agent: 81% merge rate (62/77 settled), 10 open
- github-actions bot: 100% (8/8 settled)

## Trends
- Quality: 62/100 (↑+2)
- Effectiveness: 63/100 (↑+3)
- Health: 84/100 (↓3 — CI regression #41844)
- Compiled: 253/253 (100%) ✅

## Issues Filed This Run
- 0 new issues (all known issues tracked, no new P1s warranted)

## Do Not Re-File
Code Simplifier #41842 (WIP PR #41852), Daily Safe Output Integrator #41788, BYOK Ollama #41827+#41811, CI regression #41844 (WIP PR #41849), Go Logger #41839, Agentic Audit Agent #41807, Cache Strategy #41787, Daily yamllint Fixer #41825

## Coverage Gaps (recommendations filed)
- No stale PR detection (PRs open >7d)
- No automated recovery/auto-close workflow
- No AIC budget forecasting/alerting
