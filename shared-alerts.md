# Shared Alerts — 2026-07-06T05:57Z (Workflow Health Manager)

## P0 🚨
- **Copilot BYOK stream_options (#43031):** SDK injects `stream_options` into Anthropic calls → HTTP 400. PR #43127 open for fix. **NEEDS URGENT MERGE.** DO NOT RE-FILE.

## P1 🚨
- **AI Moderator (#43352):** cache_memory_miss + no-safe-outputs. PR #43525 (fix open). DO NOT RE-FILE.
- **PR Sous Chef (#43143):** Missing `pr-processor` sub-agent. DO NOT RE-FILE.
- **PR Code Quality Reviewer (#aw_prcq_fix):** Fresh failure regression. DO NOT RE-FILE.
- **CI Integration TestMCPGateway (#42423):** Failing. DO NOT RE-FILE.
- **Sub-Agent Model Resolution Audit (#42033, #43335):** codex alpha 404, 12+ days. DO NOT RE-FILE.
- **Daily Safe Output Integrator (#42333):** Tool denial 5/5. DO NOT RE-FILE.
- **Daily BYOK Ollama (#41827):** api-proxy 503. DO NOT RE-FILE.
- **Go Logger Enhancement (#42032):** jq ARG_MAX. DO NOT RE-FILE.
- **Smoke Copilot Sub Agents (#42824):** 100% red — no model. DO NOT RE-FILE.
- **Matt Pocock Skills Reviewer (#43309):** recurring. DO NOT RE-FILE.
- **Impeccable Skills Reviewer (#43308):** engine crash (recurring). DO NOT RE-FILE.
- **Metrics Collector (#43292):** Engine failure — data stale since Jan 2026. DO NOT RE-FILE.
- **Q/E Plateau (#aw_quality_plateau):** Q:61/E:62 plateau. PR #43527 open. DO NOT RE-FILE.

## P2 ⚠️ Watch (Jul 6)
- **AR rate spike**: 22% → 60% today (early-day burst) — consistent with existing P0/sandbox systemic issues; no new root cause but bears watching through day
- **Content Moderation**: 1/5 success today (was Stable Jul 5) — pattern matches systemic AR spike, not a new root cause
- **Design Decision Gate**: AR again today (was recovered Jul 5) — intermittent, watching

## P2 ⚠️ (Existing — DO NOT RE-FILE)
- **Smoke Codex (#43281):** missing required data.
- **GitHub Remote MCP Auth Test (#43330):** failed. DO NOT RE-FILE.
- **Daily Rendering Scripts Verifier (#43353):** failed. DO NOT RE-FILE.
- **daily-experiment-report (#43355):** failed. DO NOT RE-FILE.
- **Smoke Antigravity (#43368):** no safe outputs. DO NOT RE-FILE.
- **Daily Max AI Credits Test (#43379):** failed. DO NOT RE-FILE.
- **Daily Hippo Learn (#43336):** missing required tool. DO NOT RE-FILE.
- Others: #41987,#41988,#42329,#42332,#42342,#42356,#42442,#42482,#42598,#42607,#42867,#42872,#42883,#42889,#42890,#42899,#42930,#42943,#42960

## Stable ✅
Avenger · Copilot · Copilot cloud agent · Daily Safe Outputs Git Simulator · Publish safe outputs Node image · Test Quality Sentinel · Auto-Triage Issues · Auto-Close Parent Issues · Bot Detection

## Health (Jul 6 05:57Z)
- Health 62/100 (↓ from 69 Jul 5) · P0: 1 · P1: 13 · P2: 23+
- success=24 (29%), action_required=49 (60%), skipped=8 (10%), in_progress=18, cancelled=1
- AR spike vs Jul 5 PM is notable but appears systemic (BYOK+sandbox) — no new P0/P1 root cause

## Coordination Notes
- **BYOK P0 (#43127):** Still open — priority merge; blocks large portion of AR rate
- **Q/E Plateau PR #43527:** Quality gate PR open — Campaign Manager priority
- **Codex migration (#42033):** 12+ days unactioned — escalate via campaign
- **Metrics Collector failure (#43292):** Data stale since Jan 2026 — ecosystem blind spot persists
- **AR spike Jul 6 morning**: When BYOK PR #43127 merges, expect significant AR reduction

## Do Not Re-File (cumulative through Jul 6)
#41827,#41987,#41988,#42032,#42033,#42095,#42329,#42332,#42333,#42342,#42356,#42398,#42423,#42442,#42482,#42598,#42607,#42637,#42652,#42824,#42867,#42872,#42883,#42889,#42890,#42899,#42908,#42918,#42919,#42921,#42930,#42943,#42960,#43031,#43040,#43045,#43065,#43066,#43079,#43084,#43087,#43101,#43108,#43110,#43122,#43138,#43141,#43143,#43146,#43159,#43161,#43179,#43182,#43191,#43194,#43277,#43281,#43292,#43308,#43309,#43317,#43319,#43323,#43330,#43335,#43336,#43352,#43353,#43355,#43368,#43379,#aw_ci_parser_ctx,#aw_quality_plateau,#aw_whd_jul4,#aw_whd_jul5
