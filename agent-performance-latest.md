# Agent Performance — Latest Run

**Timestamp:** 2026-06-21T13:22:00Z | **Run:** [§27905581655](https://github.com/github/gh-aw/actions/runs/27905581655)

## Summary: 57/100 Quality (→ stable) | 56/100 Effectiveness (+1) | 66/100 Health (→ stable) | AIC crisis Day 15

## Top Performers
1. Static Analysis Suite (Q:88, E:90) — 0 over-creation, built-in dedup (#40587 today)
2. CLI Version Updater (Q:86, E:92) — 5 tools updated #40592, recompiled 249/249
3. Copilot SWE Agent (Q:80, E:84) — 24/40 merged (60%) ↑ from 59% yesterday
4. Agentic Maintenance (Q:82, E:88) — 100% success
5. Team Status (Q:80, E:85) — high-quality daily summary (#40611)
6. PR Sous Chef (Q:78, E:83) — 1 failure today (#40586), was 100% yesterday (likely transient)
7. Daily Documentation Updater (Q:75, E:80) — HOLDING stable (#40606)
8. Bot Detection (Q:82, E:91) — 100% success

## Stable Recoveries
- Daily Safe Outputs Git Simulator: RECOVERED ✅ (Jun 21 — 1/1 success after Day 12+ failures)
- Daily Documentation Updater: HOLDING ✅
- Daily Workflow Updater: HOLDING ✅
- Instructions Janitor: HOLDING ✅
- Glossary Maintainer: HOLDING ✅
- Avenger: HOLDING ✅ (monitor Jun 21 for second consecutive success)

## Underperformers (Persistent)
- Code Simplifier (Q:10, E:5) Day 15: api-proxy cap + HTTP 429. #39968/#40431/#40577. PR #40578 open. DO NOT RE-FILE.
- Tool Denial Cluster (7+ workflows, Q:20, E:15): systemic Day 15+. DO NOT RE-FILE.
- Daily Model Inventory (Q:35, E:25) Day 11: session.idle. #39471. DO NOT RE-FILE.
- Daily News (Q:30, E:20) Day 8+: orphan branch signing. #40190. DO NOT RE-FILE.
- Daily Safe Output Integrator (Q:20, E:15) Day 12: tool denial. #39477. DO NOT RE-FILE.
- Daily BYOK Ollama Test (Q:30, E:20) Day 12: api-proxy cap. #39476/#40417. DO NOT RE-FILE.

## New Patterns Detected (Jun 21)
- **Auto-Triage Issues**: produced no safe outputs today (#40598) — was 100% yesterday. May be tool denial expanding.
- **Daily Hippo Learn**: no safe outputs (#40596) — watch if spreading.
- **Smoke Codex**: set_issue_field cannot bind (#40600) — temporary_id resolution bug.
- **SEC-004**: 2 handlers need body sanitization (#40594) — fix PRs already open (#40617).
- **Daily Safe Outputs Git Simulator**: ✅ RECOVERED Day 1 — hold 3 days to confirm stable.
- **Code Simplifier PR #40578**: Copilot AI fix open, awaiting pelikhan review — actionable.
- **Copilot SWE Agent**: merge rate ↑ 59%→60% (24/40 PRs, 6 open, 10 closed-unmerged).

## PR Merge Rates (Jun 21 window)
- app/copilot-swe-agent: 60% (24/40, 6 open, 10 closed-unmerged) ↑ from 59%
- app/github-actions: 50% (5/10, 4 open)

## Output Quality Sample (today)
- CLI Updater #40592: 100/100
- Static Analysis #40587: 95/100
- Team Status #40611: 90/100
- Duplicate Code Detector #40588: 88/100
- PR Triage Report #40614: 85/100

## Issues Filed This Run
- 0 new issues (all P1/P2 already tracked)
- 1 weekly discussion created

## Do Not Re-File (additions Jun 21)
- #40598 — Auto-Triage no safe outputs (monitor first)
- #40596 — Hippo Learn no safe outputs (monitor first)
