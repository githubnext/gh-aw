# Shared Alerts — 2026-07-06T13:57Z (Agent Performance Analyzer)

## P0 🚨
- **Copilot BYOK stream_options (#43031):** ✅ RESOLVED — PR #43127 MERGED Jul 4. AR spike expected to reduce. Remove from P0 on next health cycle.

## P1 🚨
- **AI Moderator (#43352):** cache_memory_miss + no-safe-outputs. PR #43525 (fix open). DO NOT RE-FILE.
- **PR Sous Chef (#43143):** Missing `pr-processor` sub-agent. DO NOT RE-FILE.
- **PR Code Quality Reviewer (#aw_prcq_fix):** Fresh failure regression. PR #43730 (bounded diff fix) MERGED Jul 6 — monitor next runs. DO NOT RE-FILE.
- **CI Integration TestMCPGateway (#42423):** Failing. DO NOT RE-FILE.
- **Sub-Agent Model Resolution Audit (#42033, #43335):** codex alpha 404, 12+ days. DO NOT RE-FILE.
- **Daily Safe Output Integrator (#42333):** Tool denial 5/5. DO NOT RE-FILE.
- **Daily BYOK Ollama (#41827):** api-proxy 503. DO NOT RE-FILE.
- **Go Logger Enhancement (#42032):** jq ARG_MAX. DO NOT RE-FILE.
- **Smoke Copilot Sub Agents (#42824):** 100% red — no model. DO NOT RE-FILE.
- **Matt Pocock Skills Reviewer (#43309):** was recurring, now 9/10 success — watch for stabilization. DO NOT RE-FILE.
- **Impeccable Skills Reviewer (#43308):** engine crash (was recurring), now 8/10 success. DO NOT RE-FILE.
- **Metrics Collector (#43292):** Engine failure — data stale since Jan 2026. DO NOT RE-FILE.
- **Q/E Plateau (#aw_quality_plateau):** Q:62/E:63 (marginal uptick). PR #43527 MERGED — observe next cycle.

## P2 ⚠️ Watch (Jul 6)
- **Content Moderation**: 11/20 success (55%) — declining from stable. Pattern matches systemic AR, but degradation noted.
- **Q workflow**: 0/16 active success — 100% AR persistent. Quality gate merged but not yet reflected.
- **Design Decision Gate**: Intermittent AR — watch.
- **PR Description Updater**: 15/20 AR — persistent underperformance.

## P2 ⚠️ (Existing — DO NOT RE-FILE)
- #41987,#41988,#42329,#42332,#42342,#42356,#42442,#42482,#42598,#42607,#42637,#42867,#42872,#42883,#42889,#42890,#42899,#42930,#42943,#42960,#43079,#43087,#43101,#43108,#43110,#43122,#43138,#43141,#43143,#43146,#43159,#43161,#43179,#43182,#43191,#43194,#43281,#43319,#43330,#43353,#43355,#43368,#43379,#43281,#43317,#43323

## Key Coordination Notes (Jul 6 13:57Z)
- BYOK P0 resolved — health manager should remove from P0 on next cycle
- Quality gate PR #43527 merged — watch Q/E scores in next 24-48h cycle
- PRCQ fix merged (#43730) — monitor PR Code Quality Reviewer success rate
- Metrics Collector (#43292) still down — analysis remains on Jul 5 PM + live run data

## Do Not Re-File (cumulative through Jul 6)
#41827,#41987,#41988,#42032,#42033,#42095,#42329,#42332,#42333,#42342,#42356,#42398,#42423,#42442,#42482,#42598,#42607,#42637,#42652,#42824,#42867,#42872,#42883,#42889,#42890,#42899,#42908,#42918,#42919,#42921,#42930,#42943,#42960,#43031,#43040,#43045,#43065,#43066,#43079,#43084,#43087,#43101,#43108,#43110,#43122,#43138,#43141,#43143,#43146,#43159,#43161,#43179,#43182,#43191,#43194,#43277,#43281,#43292,#43308,#43309,#43317,#43319,#43323,#43330,#43335,#43336,#43352,#43353,#43355,#43368,#43379,#aw_ci_parser_ctx,#aw_quality_plateau,#aw_whd_jul4,#aw_whd_jul5
