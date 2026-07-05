# Agent Performance — 2026-07-05T13:01Z | [§28741559299](https://github.com/github/gh-aw/actions/runs/28741559299)

## Scores: Q:61/100 (→ plateau wk4) | E:62/100 (→) | Health:69/100 (→)

## Top 10 Agents
| Agent | Q | E | Status |
|-------|---|---|--------|
| Copilot SWE Agent | 92 | 91 | 20 PRs merged Jul 5; 82/100 merged in 3d window |
| Auto-Triage Issues | 88 | 87 | 100% completion |
| PR Triage Agent | 88 | 86 | 100%, on-time |
| Agentic Commands | 85 | 85 | 13/14 success today |
| Content Moderation | 82 | 81 | 13/14 success today |
| Avenger | 84 | 83 | 100% |
| Auto-Close Parent Issues | 83 | 82 | 100% |
| Bot Detection | 82 | 81 | 100% |
| Claude Code Docs Review | 81 | 80 | Clean |
| Label Closed PRs | 78 | 76 | 1 action_req |

## Today's Run Summary (Jul 5, 80 runs)
- success=27 (34% ↑ from 19% Jul 4), action_required=18 (22%), skipped=28 (35%), running=7
- NOTE: higher skip rate today inflates success %; net improvement is real but partial

## New Regression Flagged
- **CGO**: 4/4 action_required today — was listed as "Confirmed Stable" in Jul 5 morning health report (05:48Z). Potential afternoon regression. NOT in DNR list. Watch for next health cycle.

## Open Positive Signals
- PR #43527 (shared quality gate — addresses Q/E plateau) OPEN, ready for review
- PR #43525 (AI Moderator noop/cache hardening) OPEN
- PR #43127 (BYOK P0 fix) OPEN — still needs urgent merge

## New Issues Filed This Run
- None (CGO flagged in shared-alerts for health manager; quality plateau already tracked #aw_quality_plateau)

## Persistent Issues (DO NOT RE-FILE)
#41827,#41987,#41988,#42032,#42033,#42095,#42329,#42332,#42333,#42342,#42356,#42398,#42423,#42442,#42482,#42598,#42607,#42637,#42652,#42824,#42867,#42872,#42883,#42889,#42890,#42899,#42908,#42918,#42919,#42921,#42930,#42943,#42960,#43031,#43040,#43045,#43065,#43066,#43079,#43084,#43087,#43101,#43108,#43110,#43122,#43138,#43141,#43143,#43146,#43159,#43161,#43179,#43182,#43191,#43194,#43277,#43281,#43292,#43308,#43309,#43317,#43319,#43323,#43330,#43335,#43336,#43352,#43353,#43355,#43368,#43379,#aw_ci_parser_ctx,#aw_quality_plateau,#aw_whd_jul4

## Key Findings (Jul 5 PM)
- Q/E plateau now in week 4 — PR #43527 (quality gate) is the active mitigation; needs merge
- Copilot SWE Agent continues elite output (20 merges today, Q:92/E:91 steady)
- CGO regression: 4/4 action_req — was stable per Jul 5 morning health report; flag for next health cycle
- CWI (2/2 action_req) and Doc Build-Deploy (2/2 action_req) may warrant monitoring
- BYOK P0 (#43127): still unmerged — highest-priority action needed
- Metrics Collector still failing (#43292) → data collection remains stale (Jan 2026 baseline only)
