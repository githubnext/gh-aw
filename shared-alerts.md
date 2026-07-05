# Shared Alerts — 2026-07-05T05:48Z (Workflow Health Manager)

## P0 🚨
- **Copilot BYOK stream_options (#43031):** SDK injects `stream_options` into Anthropic calls → HTTP 400. PR #43127 open for fix. DO NOT RE-FILE.

## P1 🚨
- **AI Moderator (#43352):** cache_memory_miss + no-safe-outputs. DO NOT RE-FILE.
- **PR Sous Chef (#43143):** Missing `pr-processor` sub-agent. DO NOT RE-FILE.
- **PR Code Quality Reviewer (#aw_prcq_fix):** Fresh failure regression. DO NOT RE-FILE.
- **CI Integration context import (#aw_ci_parser_ctx):** RESOLVED ✅ via #43323 merged Jul 4.
- **CI Integration TestMCPGateway (#42423):** Failing. DO NOT RE-FILE.
- **Sub-Agent Model Resolution Audit (#42033, #43335):** codex alpha 404, 10+ days. DO NOT RE-FILE.
- **Daily Safe Output Integrator (#42333):** Tool denial 5/5. DO NOT RE-FILE.
- **Daily BYOK Ollama (#41827):** api-proxy 503. DO NOT RE-FILE.
- **Go Logger Enhancement (#42032):** jq ARG_MAX. DO NOT RE-FILE.
- **Smoke Copilot Sub Agents (#42824):** 100% red — no model. DO NOT RE-FILE.
- **Matt Pocock Skills Reviewer (#43309):** 4th failure in 7 days. DO NOT RE-FILE.
- **Impeccable Skills Reviewer (#43308):** engine crash (recurring). DO NOT RE-FILE.
- **Metrics Collector (#43292):** Engine failure — analysis blind spot. DO NOT RE-FILE.
- **Q/E Plateau (#aw_quality_plateau):** Q:61/E:62 for 3+ weeks. DO NOT RE-FILE.

## P2 ⚠️ (DO NOT RE-FILE)
- **Smoke Codex (#43281):** missing required data.
- **Design Decision Gate (#43319):** recovered today ✅
- **GitHub Remote MCP Auth Test (#43330):** failed Jul 4. DO NOT RE-FILE.
- **Daily Rendering Scripts Verifier (#43353):** failed Jul 4. DO NOT RE-FILE.
- **daily-experiment-report (#43355):** failed Jul 4. DO NOT RE-FILE.
- **Smoke Antigravity (#43368):** no safe outputs (recurring #43087). DO NOT RE-FILE.
- **Daily Max AI Credits Test (#43379):** failed Jul 4. DO NOT RE-FILE.
- **Daily Hippo Learn (#43336):** missing required tool. DO NOT RE-FILE.
- Others: #41987,#41988,#42329,#42332,#42342,#42356,#42442,#42482,#42598,#42607,#42867,#42872,#42883,#42889,#42890,#42899,#42930,#42943,#42960

## Stable ✅
Copilot SWE Agent · Auto-Triage Issues · PR Triage Agent · Avenger · Content Moderation · Auto-Close Parent Issues · Bot Detection · Claude Code Docs Review · Design Decision Gate (recovered Jul 5) · Test Quality Sentinel · CJS · CGO

## Health (Jul 5 early data)
- Health 69/100 (→ stable) · P0: 1 · P1: 12 · P2: 20+
- First 50 runs today: action_required=23, success=15, running=12, no hard failures
- action_required systemic pressure continues (~46% of sampled runs)

## Coordination Notes
- **Q/E plateau alert:** #aw_quality_plateau filed Jul 4 — Campaign Manager should prioritize prompt improvement campaign
- **Codex migration (#42033):** Still unactioned 11+ days — needs repair campaign escalation
- **BYOK P0 (#43031):** PR #43127 still open — needs urgent merge
- **Metrics Collector failure (#43292):** Data collection blind spot; metrics stale since Jan 2026
- **No new issues filed Jul 5 run** — all tracked patterns already have open issues
- **Design Decision Gate recovered** today after Jul 4 failure — may be transient

## Do Not Re-File (cumulative through Jul 5)
#41827,#41987,#41988,#42032,#42033,#42095,#42329,#42332,#42333,#42342,#42356,#42398,#42423,#42442,#42482,#42598,#42607,#42637,#42652,#42824,#42867,#42872,#42883,#42889,#42890,#42899,#42908,#42918,#42919,#42921,#42930,#42943,#42960,#43031,#43040,#43045,#43065,#43066,#43079,#43084,#43087,#43101,#43108,#43110,#43122,#43138,#43141,#43143,#43146,#43159,#43161,#43179,#43182,#43191,#43194,#43277,#43281,#43292,#43308,#43309,#43317,#43319,#43323,#43330,#43335,#43336,#43352,#43353,#43355,#43368,#43379,#aw_ci_parser_ctx,#aw_quality_plateau,#aw_whd_jul4
