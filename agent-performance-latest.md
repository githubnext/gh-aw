# Agent Performance — 2026-07-06T13:57Z | [§28796809903](https://github.com/github/gh-aw/actions/runs/28796809903)

## Scores: Q:62/100 (↑ from 61 wk4) | E:63/100 (↑ from 62) | Health:62/100 (→)

## Key PR Merges Since Last Run
- **PR #43127 (BYOK fix)** — MERGED Jul 4 ✅ (was listed as open in Jul 5 report)
- **PR #43527 (quality gate)** — MERGED Jul 5 ✅
- **PR #43730 (PRCQ bounded diff)** — MERGED Jul 6 ✅

## Top 10 Agents (Jul 6)
| Agent | Q | E | Runs | Eff% | Status |
|-------|---|---|------|------|--------|
| Copilot SWE Agent | 92 | 91 | 37 PRs/7d | 65% merge | 24/37 merged 7d |
| Auto-Triage Issues | 88 | 87 | 20/20 | 100% | Stable ✅ |
| PR Triage Agent | 88 | 86 | 19/20 | 95% | Stable ✅ |
| Auto-Close Parent Issues | 83 | 82 | 10/10 | 100% | Stable ✅ |
| Issue Monster | 80 | 80 | 10/10 | 100% | Stable ✅ |
| Avenger | 84 | 83 | 9/10 | 90% | Stable ✅ |
| Matt Pocock Skills Reviewer | 70 | 68 | 9/10 | 90% | Improving (was chronic fail) |
| Impeccable Skills Reviewer | 68 | 65 | 8/10 | 80% | Improving |
| Content Moderation | 65 | 62 | 11/20 | 55% | ⚠ Declining (P2 watch) |
| PR Code Quality Reviewer | 60 | 58 | 6/10 | 60% | ⚠ Mixed (PR #43730 fix merged today) |

## Bottom Agents (Jul 6)
| Agent | Q | E | Runs | Eff% | Issue |
|-------|---|---|------|------|-------|
| Q workflow | 30 | 25 | 0/16 | 0% | 100% AR — plateau persists |
| Agentic Commands | 40 | 38 | 4/20 | 20% | Persistent AR — systemic |
| PR Description Updater | 45 | 42 | 4/20 | 20% | 15/20 AR |
| Design Decision Gate | 48 | 45 | ~50% | ~50% | Intermittent AR |
| Deployment Incident Monitor | 50 | 40 | 1/10 | 90% skip | Near-inactive (correct skip behavior) |

## New Findings (Jul 6)
- **BYOK P0 (#43031)** already resolved (PR #43127 merged Jul 4) — shared-alerts was stale
- **Quality gate (#43527)** already merged — Q/E may show improvement in next cycle
- **PRCQ improvement** (#43730 merged Jul 6) — should reduce PRCQ 3/10 failure rate
- **AR spike (22%→60%)** on Jul 6 morning partially attributable to now-merged BYOK fix

## New Issues Filed This Run
- None (all patterns already tracked in DNR list)

## Persistent Issues (DO NOT RE-FILE)
#41827,#41987,#41988,#42032,#42033,#42095,#42329,#42332,#42333,#42342,#42356,#42398,#42423,#42442,#42482,#42598,#42607,#42637,#42652,#42824,#42867,#42872,#42883,#42889,#42890,#42899,#42908,#42918,#42919,#42921,#42930,#42943,#42960,#43031,#43040,#43045,#43065,#43066,#43079,#43084,#43087,#43101,#43108,#43110,#43122,#43138,#43141,#43143,#43146,#43159,#43161,#43179,#43182,#43191,#43194,#43277,#43281,#43292,#43308,#43309,#43317,#43319,#43323,#43330,#43335,#43336,#43352,#43353,#43355,#43368,#43379,#aw_ci_parser_ctx,#aw_quality_plateau,#aw_whd_jul4,#aw_whd_jul5
