# Workflow Health — 2026-07-05T05:48Z

Score: 69/100 (→ flat from Jul 4) | Run: §28731084381

## KEY FINDINGS

### Compilation Status
- **258/258 workflows have lock files (100% ✅)**. Compile-validate clean.

### P0 Issues (DO NOT RE-FILE)
- **Copilot BYOK stream_options (#43031 OPEN)**: SDK injects OpenAI-only `stream_options` into Anthropic calls → HTTP 400. PR #43127 open for fix. DO NOT RE-FILE.

### P1 Resolved ✅
- **CI Integration Parser context import (#aw_ci_parser_ctx)**: Fixed via #43323 merged Jul 4.

### P1 Issues (DO NOT RE-FILE)
- **PR Sous Chef** (#43040, #43143): Missing `pr-processor` sub-agent. DO NOT RE-FILE.
- **Smoke CI** (#42398): EACCES mkdir /tmp/gh-aw (action_required continuing). DO NOT RE-FILE.
- **CI Integration TestMCPGateway** (#42423): Failing. DO NOT RE-FILE.
- **Sub-Agent Model Resolution** (#42033, #43335): codex alpha 404. DO NOT RE-FILE.
- **Daily Safe Output Integrator** (#42333): Tool denial 5/5. DO NOT RE-FILE.
- **Daily BYOK Ollama** (#41827): api-proxy 503. DO NOT RE-FILE.
- **Go Logger Enhancement** (#42032): jq ARG_MAX. DO NOT RE-FILE.
- **Smoke Copilot Sub Agents** (#42824): 100% red - no model. DO NOT RE-FILE.
- **AI Moderator** (#43352): cache_memory_miss + no-safe-outputs. DO NOT RE-FILE.
- **PR Code Quality Reviewer** (#aw_prcq_fix): fresh failure regression. DO NOT RE-FILE.
- **Matt Pocock Skills Reviewer** (#43309): 4th failure in 7 days. DO NOT RE-FILE.
- **Impeccable Skills Reviewer** (#43308): engine crash (recurring). DO NOT RE-FILE.
- **Metrics Collector** (#43292): Engine failure — analysis blind spot. DO NOT RE-FILE.

### P2 Issues (Previously Tracked — DO NOT RE-FILE)
#41987,#41988,#42329,#42332,#42342,#42356,#42442,#42482,#42598,#42607,#42637,#42867,#42870,#42871,#42883,#42889,#42890,#42899,#42908,#42918,#42919,#42930,#42943,#42960,#43079,#43087,#43101,#43108,#43110,#43122,#43138,#43141,#43143,#43146,#43159,#43161,#43179,#43182,#43191,#43194,#43281,#43319,#43330,#43353,#43355,#43368,#43379

### Confirmed Stable ✅
- Design Decision Gate (recovered today ✅) | Avenger | Auto-Close Parent Issues | Safe Output Health Monitor | CI (partial) | Bot Detection | Content Moderation | Auto-Triage Issues

### Systemic Issues (unchanged)
1. **Copilot BYOK 400** (stream_options mismatch) → #43031 (P0)
2. **Sandbox seccomp restrictions** (blocks node/npm/go binary exec) → #43110, #43101
3. **Smoke CI sandbox EACCES** → #42398
4. **Codex alpha 404** (9+ workflows, 10+ days) → #42033, #43335
5. **action_required systemic pressure** (~46% of today's first 50 runs)

### Run Summary (Jul 5 first ~6 min of day)
- action_required: 23 | success: 15 | running: 12 | No hard failures today ✅
- No new P0/P1 issues detected
- Metrics collection still stale (Jan 2026 data — Metrics Collector still broken #43292)

### Actions Taken This Run
- No new issues filed (all patterns already tracked)
- Shared memory updated

### Do Not Re-File (cumulative through Jul 5)
#41827,#41987,#41988,#42032,#42033,#42095,#42329,#42332,#42333,#42342,#42356,#42398,#42423,#42442,#42482,#42598,#42607,#42637,#42652,#42824,#42867,#42872,#42883,#42889,#42890,#42899,#42908,#42918,#42919,#42921,#42930,#42943,#42960,#43031,#43040,#43045,#43065,#43066,#43079,#43084,#43087,#43101,#43108,#43110,#43122,#43138,#43141,#43143,#43146,#43159,#43161,#43179,#43182,#43191,#43194,#43277,#43281,#43292,#43308,#43309,#43317,#43319,#43323,#43330,#43335,#43336,#43352,#43353,#43355,#43368,#43379,#aw_ci_parser_ctx,#aw_quality_plateau,#aw_whd_jul4
