# Agent Performance — 2026-07-08T13:26Z | [§28945138633](https://github.com/github/gh-aw/actions/runs/28945138633)

## Scores: Q:63/100 (↑ from 62 Jul 7) | E:64/100 (↑ from 63) | Health:63/100 (↓ from 65)

## Key PR Merges Since Last Run (Jul 7-8)
- **PR #44286 (sink-visibility default fix)** — MERGED Jul 8 ✅
- **PR #44277 (StripANSI unit tests)** — MERGED Jul 8 ✅
- **PR #44270 (skip lockdown autodetect)** — MERGED Jul 8 ✅
- **PR #44268 (eslint no-github-request-interpolated-route)** — MERGED Jul 8 ✅
- **PR #44256 (eslint-factory try/finally fix)** — MERGED Jul 8 ✅
- **PR #44255 (replace deprecated activation outputs)** — MERGED Jul 8 ✅
- **PR #44245 (sink-visibility runtime compute)** — MERGED Jul 8 ✅
- **PR #44240 (eslint-factory destructured fs)** — MERGED Jul 8 ✅
- **PR #44238 (workflow-edit guard for stale locks)** — MERGED Jul 8 ✅
- **PR #44214 (restore comment-memory config)** — MERGED Jul 8 ✅
- **PR #44209 (designer drift audit fix)** — MERGED Jul 8 ✅

## Top 10 Agents (Jul 8)
| Agent | Q | E | Runs | Eff% | Status |
|-------|---|---|------|------|--------|
| Copilot SWE Agent | 92 | 91 | 11 PRs merged today | ~73% merge | Active ✅ |
| Auto-Triage Issues | 88 | 87 | 1/1 | 100% | Stable ✅ |
| PR Triage Agent | 88 | 86 | 1/1 | 100% | Stable ✅ |
| Content Moderation | 80 | 78 | 5/5 | 100% | Recovered ✅ |
| Agentic Maintenance | 83 | 82 | 1/1 | 100% | Stable ✅ |
| Running Copilot Code Review | 80 | 80 | 3/3 | 100% | Stable ✅ |
| Smoke CI | 78 | 75 | 4/4 | 100% | Improved ✅ |
| CGO | 72 | 70 | 4/5 | 80% | Improving ↑ |
| CWI | 72 | 70 | 3/3 | 100% | Recovered ✅ |
| Avenger | 84 | 83 | stable | ~100% | Stable ✅ |

## Bottom Agents (Jul 8)
| Agent | Q | E | Runs | Eff% | Issue |
|-------|---|---|------|------|-------|
| Q workflow | 25 | 15 | 0/24 ok | 0% (79% AR) | 100% AR — quality gate PR #43527 not yet merged |
| AI Moderator | 35 | 30 | 0/5 | 0% (skip) | Engine failure (Codex) — #44241 filed |
| PR Code Quality Reviewer | 38 | 35 | 0/1 ok | 0% | CLI hang-on-exit — #aw_pr_cq_tqs filed TODAY |
| Test Quality Sentinel | 38 | 35 | 0/2 ok | 0% | CLI hang-on-exit — #aw_pr_cq_tqs filed TODAY |
| Agentic Commands | 40 | 38 | 1/24 ok | 8% (79% AR) | Persistent AR — #43079 existing |
| Design Decision Gate | 48 | 45 | 0/1 ok | 0% | Redesign/deprecation candidate |
| Matt Pocock Skills Reviewer | 50 | 48 | 0/1 ok | 0% | Deprecation candidate |
| Impeccable Skills Reviewer | 55 | 52 | 0/1 ok | 0% | CLI hang — fix PR #44254 open |

## New Findings (Jul 8)
- **PR Code Quality Reviewer + Test Quality Sentinel**: NEW 100% AR cluster — CLI hang-on-exit same as Impeccable. Issue #aw_pr_cq_tqs filed.
- **CGO stabilizing**: 1/5 AR today (was 3/3 AR earlier AM) — WHM monitoring #38777
- **Content Moderation recovered**: 5/5 success today (was 50% mixed earlier this week)
- **Discussion created**: Weekly performance report (Jul 8)

## New Issues Filed This Run
- #aw_pr_cq_tqs — PR Code Quality Reviewer + Test Quality Sentinel 100% AR (CLI hang)

## Persistent Issues (DO NOT RE-FILE)
#41827,#41987,#41988,#42032,#42033,#42095,#42329,#42332,#42333,#42342,#42356,#42398,#42423,#42442,#42482,#42598,#42607,#42637,#42652,#42824,#42867,#42872,#42883,#42889,#42890,#42899,#42908,#42918,#42919,#42921,#42930,#42943,#42960,#43031,#43040,#43045,#43065,#43066,#43079,#43084,#43087,#43101,#43108,#43110,#43122,#43138,#43141,#43143,#43146,#43159,#43161,#43179,#43182,#43191,#43194,#43277,#43281,#43292,#43308,#43309,#43317,#43319,#43323,#43330,#43335,#43336,#43352,#43353,#43355,#43368,#43379,#43883,#43894,#43895,#43925,#43927,#43930,#44006,#44016,#44031,#44032,#44035,#44241,#aw_ai_mod_jul8,#aw_ci_parser_ctx,#aw_pr_cq_tqs,#aw_quality_plateau,#aw_whd_jul4,#aw_whd_jul5,#aw_whd_jul6,#aw_whd_jul7
