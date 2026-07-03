# Agent Performance — 2026-07-03T13:18Z | [§28662997895](https://github.com/github/gh-aw/actions/runs/28662997895)

## Scores: Q:61/100 (→) | E:62/100 (→) | Health:72/100 (→ unchanged from Jul 2)

## Top 10 Agents
| Agent | Q | E | Status |
|-------|---|---|--------|
| Copilot SWE Agent | 92 | 91 | 86% merge rate (↑3%), 31/36 resolved |
| Auto-Triage Issues | 88 | 87 | 100% (2/2) |
| PR Triage Agent | 88 | 86 | 100%, #43190 on-time |
| Avenger | 84 | 83 | 100% |
| Content Moderation | 82 | 81 | 100% (2/2) |
| Auto-Close Parent Issues | 83 | 82 | 100% |
| Bot Detection | 82 | 81 | 100% |
| Claude Code Docs Review | 81 | 80 | Clean #43193 |
| Label Closed PRs | 78 | 76 | 1 action_req (expected) |
| PR Description Updater | 77 | 76 | 1 success, 1 action_req |

## Failures Today (9)
- Smoke CI (#43122, #42398 open)
- Test Quality Sentinel (§28661680941)
- PR Code Quality Reviewer (§28661680929, #aw_prcq_fix NEW)
- CJS failure
- Matt Pocock Skills Reviewer (#43191)
- Impeccable Skills Reviewer (#43079)
- Design Decision Gate
- Doc Build - Deploy
- Daily Max Ai Credits Test (#43182)

## New Issues Filed This Run
- **#aw_ai_mod_fix**: AI Moderator systemic cache_memory + no-safe-outputs fix
- **#aw_prcq_fix**: PR Code Quality Reviewer fresh failure regression tracking

## P1 Persistent (DO NOT RE-FILE)
#42033 · #42032 · #41827 · #42333 · #42398 · #42423 · #42824 · #43031 · #43040(closed?) · #43079 · #43087 · #43101 · #43108(closed) · #43110 · #43122 · #43138 · #43141 · #43143 · #43146 · #43159 · #43161 · #43179 · #43182 · #43191 · #43194

## Key Findings (Jul 3)
- AI Moderator: dual failure modes confirmed → #aw_ai_mod_fix created
- PR Sous Chef: failure mode shifted from HTTP 400 → missing `pr-processor` tool (#43143)
- PR Code Quality Reviewer: #42095 closed but fresh failure today → #aw_prcq_fix
- Copilot SWE Agent: PR merge rate hit 86% (best since tracking)
- WHM recovered today (run §28640918061) after Jul 2 outage
- BYOK P0 (#43031): PR #43127 (fix/detect stream_options HTTP 400) open for review
- Codex migration still not actioned (9 workflows, 10+ days failing)

## Do Not Re-File
#41827,#41987,#41988,#42032,#42033,#42095,#42329,#42332,#42333,#42342,#42356,#42398,#42423,#42442,#42482,#42598,#42607,#42637,#42824,#42867,#42872,#42883,#42889,#42890,#42899,#42908,#42918,#42919,#42921,#42930,#42943,#42960,#43031,#43040,#43079,#43087,#43101,#43108,#43110,#43122,#43138,#43141,#43143,#43146,#43159,#43161,#43179,#43182,#43191,#43194
