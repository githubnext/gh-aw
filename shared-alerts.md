# Shared Alerts — 2026-07-07T05:50Z (Workflow Health Manager)

## P0 🚨 (Monitoring)
- **Copilot BYOK stream_options (#43031):** PR #43127 MERGED Jul 4. AR rate dropping (60% → 39%). Commented on #43031 recommending closure. Monitor for full resolution.

## P1 🚨
- **AI Moderator (#43925):** New issue (old #43352 closed). Still failing. cache_memory_miss + no-safe-outputs. DO NOT RE-FILE.
- **PR Sous Chef (#43143):** Missing `pr-processor` sub-agent. DO NOT RE-FILE.
- **Smoke CI (#42398, #43908):** EACCES mkdir. DO NOT RE-FILE.
- **CI Integration TestMCPGateway (#42423):** Failing. DO NOT RE-FILE.
- **Sub-Agent Model Resolution Audit (#42033, #43335):** codex alpha 404. DO NOT RE-FILE.
- **Daily Safe Output Integrator (#42333):** Tool denial 5/5. DO NOT RE-FILE.
- **Daily BYOK Ollama (#41827, #43883):** api-proxy 503. DO NOT RE-FILE.
- **Go Logger Enhancement (#42032):** jq ARG_MAX. DO NOT RE-FILE.
- **Smoke Copilot Sub Agents (#42824):** 100% red — no model. DO NOT RE-FILE.
- **Matt Pocock Skills Reviewer (#43309, #43894):** recurring. DO NOT RE-FILE.
- **Impeccable Skills Reviewer (#43308, #43895):** engine crash pattern. DO NOT RE-FILE.
- **Metrics Collector (#43292):** Engine failure — data stale since Jan 2026. DO NOT RE-FILE.
- **Daily yamllint Fixer (#43927):** 3/10 success (30%). Recurring failures. DO NOT RE-FILE.
- **Code Simplifier (#43930):** 6/10 success (60%), alternating pattern. DO NOT RE-FILE.

## P2 ⚠️ Watch (Jul 7)
- **Content Moderation**: 7/15 success (47%) today — mixed AR/success batches. Watch for stabilization.
- **Q workflow**: 0/active success (100% AR) — quality gate PR #43527 merged, awaiting impact.
- **CGO/CJS/CWI**: Single AR runs today — isolated events, no issue warranted.

## P2 ⚠️ (Existing — DO NOT RE-FILE)
- #41987,#41988,#42329,#42332,#42342,#42356,#42442,#42482,#42598,#42607,#42637,#42867,#42872,#42883,#42889,#42890,#42899,#42930,#42943,#42960,#43079,#43087,#43101,#43108,#43110,#43122,#43138,#43141,#43143,#43146,#43159,#43161,#43179,#43182,#43191,#43194,#43281,#43319,#43330,#43353,#43355,#43368,#43379,#43281,#43317,#43323

## Key Coordination Notes (Jul 7 05:50Z)
- BYOK fix (#43127) measurably improving AR rate (60% → 39%) — health improving
- AI Moderator needs attention: PR #43525 merged Jul 5 but still failing
- yamllint Fixer and Code Simplifier now have fresh issues — newly tracked
- Quality gate PR #43527 merged — watch Q/E scores next 24-48h
- Metrics Collector (#43292) still down — analysis on live run data only

## Do Not Re-File (cumulative through Jul 7)
#41827,#41987,#41988,#42032,#42033,#42095,#42329,#42332,#42333,#42342,#42356,#42398,#42423,#42442,#42482,#42598,#42607,#42637,#42652,#42824,#42867,#42872,#42883,#42889,#42890,#42899,#42908,#42918,#42919,#42921,#42930,#42943,#42960,#43031,#43040,#43045,#43065,#43066,#43079,#43084,#43087,#43101,#43108,#43110,#43122,#43138,#43141,#43143,#43146,#43159,#43161,#43179,#43182,#43191,#43194,#43277,#43281,#43292,#43308,#43309,#43317,#43319,#43323,#43330,#43335,#43336,#43352,#43353,#43355,#43368,#43379,#43883,#43894,#43895,#43925,#43927,#43930,#aw_ci_parser_ctx,#aw_quality_plateau,#aw_whd_jul4,#aw_whd_jul5,#aw_whd_jul6
