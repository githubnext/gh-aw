# Agent Performance — 2026-07-07T13:26Z | [§28869597955](https://github.com/github/gh-aw/actions/runs/28869597955)

## Scores: Q:63/100 (↑ from 62 Jul 6) | E:64/100 (↑ from 63) | Health:65/100 (↑)

## Key PR Merges Since Last Run (Jul 6-7)
- **PR #43967 (osgetenvlibrary lint suppression)** — MERGED Jul 7 ✅
- **PR #43957 (AWF firewall v0.27.26)** — MERGED Jul 7 ✅
- **PR #43950 (yamllint Fixer revert to Claude)** — MERGED Jul 7 ✅
- **PR #43940 (jobs.generated.needs augment)** — MERGED Jul 7 ✅
- **PR #43939 (5 linter migrations)** — MERGED Jul 7 ✅

## Top 10 Agents (Jul 7)
| Agent | Q | E | Runs | Eff% | Status |
|-------|---|---|------|------|--------|
| Copilot SWE Agent | 92 | 91 | 5 PRs merged (7) | ~65% merge | Active ✅ |
| Auto-Triage Issues | 88 | 87 | 1/1 | 100% | Stable ✅ |
| PR Triage Agent | 88 | 86 | 1/1 | 100% | Stable ✅ |
| Avenger | 84 | 83 | 2/2 | 100% | Stable ✅ |
| Agentic Maintenance | 83 | 82 | 1/1 | 100% | Stable ✅ |
| Bot Detection | 83 | 82 | 1/1 | 100% | Stable ✅ |
| CWI | 75 | 74 | 3/4 | 75% | Stable ✅ |
| Content Moderation | 65 | 62 | 2/4 | 50% | ⚠ Mixed (watch) |
| Agentic Commands | 40 | 38 | 3/10 | 30% | ⚠ Persistent AR (7/10 today) |
| Matt Pocock Skills Reviewer | 70 | 68 | improving | ~90% | Improving |

## Bottom Agents (Jul 7)
| Agent | Q | E | Runs | Eff% | Issue |
|-------|---|---|------|------|-------|
| Q workflow | 30 | 25 | 0/7 ok | 0% | 100% AR — quality gate not yet impacting |
| AI Moderator | 35 | 30 | 0/4 ok | 0% | cache_memory_miss + no-safe-outputs |
| Smoke CI | 38 | 35 | 0/5 ok | 0% | 2 fail + 2 AR (EACCES sandbox) |
| CGO | 38 | 35 | 0/4 ok | 0% | 2 fail + 1 AR — worsening today |
| PR Sous Chef | 42 | 40 | 0/1 ok | 0% | Missing pr-processor (#43143) |
| Design Decision Gate | 48 | 45 | 0/1 ok (skip) | 0% | Intermittent skip |

## New Findings (Jul 7)
- **Smoke AOAI (Entra + apikey)**: New incomplete/missing-tool issues filed today (#44031, #44032, #44035)
- **Daily Max Ai Credits Test failed**: New failure issue #44016 filed
- **CGO worsening**: 2 failures + 1 AR today (was 1 AR Jul 7 AM) — WHM should monitor
- **AB Advisor active**: Filed 2 experiment campaign issues (#44011, #44012) — healthy output
- **yamllint Fixer**: Reverted to Claude (#43950 merged) — should improve success rate from 30%

## New Issues Filed This Run
- None (all patterns tracked; CGO worsening noted below)

## Persistent Issues (DO NOT RE-FILE)
#41827,#41987,#41988,#42032,#42033,#42095,#42329,#42332,#42333,#42342,#42356,#42398,#42423,#42442,#42482,#42598,#42607,#42637,#42652,#42824,#42867,#42872,#42883,#42889,#42890,#42899,#42908,#42918,#42919,#42921,#42930,#42943,#42960,#43031,#43040,#43045,#43065,#43066,#43079,#43084,#43087,#43101,#43108,#43110,#43122,#43138,#43141,#43143,#43146,#43159,#43161,#43179,#43182,#43191,#43194,#43277,#43281,#43292,#43308,#43309,#43317,#43319,#43323,#43330,#43335,#43336,#43352,#43353,#43355,#43368,#43379,#43883,#43894,#43895,#43925,#43927,#43930,#44006,#44016,#44031,#44032,#44035,#aw_ci_parser_ctx,#aw_quality_plateau,#aw_whd_jul4,#aw_whd_jul5,#aw_whd_jul6,#aw_whd_jul7
