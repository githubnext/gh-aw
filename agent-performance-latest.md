# Agent Performance — Latest Run

**Timestamp:** 2026-06-19T13:52:00Z | **Run:** [§27829611941](https://github.com/github/gh-aw/actions/runs/27829611941)

## Summary: 57/100 Quality (+0) | 55/100 Effectiveness (+0) | 66/100 Health (↓1) | AIC crisis Day 13

## Top Performers
1. copilot-swe-agent (Q:80, E:82) — 31 PRs this period, 51% merge rate (8 still open/pending review)
2. Bot Detection (Q:82, E:91) — 100% success today
3. Agentic Maintenance (Q:82, E:88) — 100% success today
4. Auto-Triage Issues (Q:81, E:84) — 100% success today
5. Content Moderation (Q:76, E:80) — 100% success today
6. Smoke Test Suite (Q:75, E:85) — healthy lifecycle, 13+ issues created/closed same day
7. Daily Docs/Glossary/Instructions (Q:80, E:85) — holding recovery, 100% PR merge rate
8. Failure Investigator (Q:78, E:82) — filed Daily News orphan analysis #40190

## Stable Recoveries (Holding)
- Daily Documentation Updater: HOLDING ✅
- Daily Workflow Updater: HOLDING ✅
- Instructions Janitor: HOLDING ✅
- Glossary Maintainer: HOLDING ✅

## Underperformers (Persistent)
- Code Simplifier (Q:10, E:5) Day 13: api-proxy cap + HTTP 429. #39968. DO NOT RE-FILE.
- Tool Denial Cluster (7+ workflows, Q:20, E:15): systemic. DO NOT RE-FILE.
- Daily Model Inventory (Q:35, E:25) Day 11: session.idle. DO NOT RE-FILE.
- Daily News (Q:30, E:20) Day 8+: orphan branch signing. DO NOT RE-FILE.
- Daily Safe Output Integrator (Q:20, E:15) Day 12: tool denial. DO NOT RE-FILE.
- Daily BYOK Ollama Test (Q:30, E:20) Day 12: transient_bad_request. DO NOT RE-FILE.
- Avenger (Q:50, E:40): new failure today (run 27828994297); tracked in #40145. DO NOT RE-FILE.

## New Patterns Detected (Jun 19)
- **Avenger regression**: Failed today (run 27828994297) — ERR_CONFIG log parse; existing issue #40145 tracks this.
- **LintMonster continuing**: 3 new issues today (#40210-40212); alternating pattern per #39511.
- **Token-consumption double-issue**: 2 issues per day (04:54 + 13:10) — both scheduled runs; first closes same day. Normal behavior, not a bug.

## PR Merge Rates (Jun 19 window)
- copilot-swe-agent: 51% (16/31 PRs, 8 still open — pending review)
- app/github-actions (automated): 70% (7/10)
- Docs/Glossary/Instructions: 100% (post-recovery)

## Issues Filed This Run
- 0 new issues (all P1s already tracked; Avenger has #40145; no new systemic issues)

## Do Not Re-File (additions Jun 19)
- Avenger failure (#40145 covers current failure mode)
