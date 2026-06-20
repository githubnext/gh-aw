# Agent Performance — Latest Run

**Timestamp:** 2026-06-20T13:52:00Z | **Run:** [§27872116337](https://github.com/github/gh-aw/actions/runs/27872116337)

## Summary: 57/100 Quality (+0) | 55/100 Effectiveness (+0) | 66/100 Health (→ stable) | AIC crisis Day 14

## Top Performers
1. Static Analysis Suite (Q:88, E:90) — 0 over-creation, built-in dedup
2. CLI Version Updater (Q:86, E:92) — 5 tools updated, 249/249 recompiled (#40445)
3. Copilot SWE Agent (Q:80, E:82) — 22 PRs, 59% merge rate ↑ from 51% (6 open)
4. Bot Detection (Q:82, E:91) — 100% success
5. Agentic Maintenance (Q:82, E:88) — 100% success today
6. Auto-Triage Issues (Q:81, E:84) — 100% success, no over-creation
7. Issue Monster (Q:80, E:80) — 100% today (5/5)
8. PR Sous Chef (Q:78, E:83) — 100% today (5/5)
9. Team Status (Q:80, E:85) — high-quality daily summary (#40461)

## Stable Recoveries (Holding)
- Avenger: RECOVERED ✅ (100%, 4/4 — was mixed Jun 19)
- Daily Documentation Updater: HOLDING ✅
- Daily Workflow Updater: HOLDING ✅
- Instructions Janitor: HOLDING ✅
- Glossary Maintainer: HOLDING ✅

## Underperformers (Persistent)
- Code Simplifier (Q:10, E:5) Day 14: api-proxy cap + HTTP 429. #39968/#40431. DO NOT RE-FILE.
- Tool Denial Cluster (7+ workflows, Q:20, E:15): systemic Day 14+. DO NOT RE-FILE.
- Daily Model Inventory (Q:35, E:25) Day 11: session.idle. #39471. DO NOT RE-FILE.
- Daily News (Q:30, E:20) Day 8+: orphan branch signing. #40190. DO NOT RE-FILE.
- Daily Safe Output Integrator (Q:20, E:15) Day 12: tool denial. #39477. DO NOT RE-FILE.
- Daily BYOK Ollama Test (Q:30, E:20) Day 12: api-proxy cap. #39476/#40417. DO NOT RE-FILE.

## New Patterns Detected (Jun 20)
- **Avenger RECOVERED**: 100% today (4/4) after ERR_CONFIG regression Jun 19. Watch for stability Jun 21.
- **aw-failures duplicate rate**: #40417 (BYOK Ollama) and #40431 (Code Simplifier) re-filed today — already tracked. ~30-40% duplicate rate.
- **Skillet startup failures**: #40447 filed today (27/27 failures on push events). Expected behavior for new slash-command workflow.
- **copilot-swe-agent improvement**: merge rate ↑ 51%→59% (13/22 PRs).
- **Content Moderation degraded**: 25% today (1/4) — monitor Jun 21.

## PR Merge Rates (Jun 20 window)
- copilot-swe-agent: 59% (13/22, 6 open, 3 closed-unmerged) ↑ from 51%
- app/github-actions: 43% (3/7, 4 open)

## Output Quality Sample (today)
- CLI Updater #40445: 100/100
- Static Analysis #40440: 95/100
- Ambient Context #40385: 90/100
- Team Status #40461: 90/100
- Skillet Failure #40447: 85/100

## Issues Filed This Run
- 0 new issues (all P1/P2 already tracked)
- 1 weekly discussion created

## Do Not Re-File (additions Jun 20)
- #40417 — BYOK Ollama re-filed by aw-failures (same as #39476)
- #40431 — Code Simplifier re-filed by aw-failures (same as #39968)
