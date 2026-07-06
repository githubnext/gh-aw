# Workflow Health — 2026-07-06T05:57Z

Score: 62/100 (↓ from 69 Jul 5) | Run: §28770848460

## KEY FINDINGS

### Compilation Status
- **258/258 workflows have lock files (100% ✅)**. Compile-validate clean.

### Health Overview (Jul 6 early-day)
- Total: 258 | Healthy: 5 | Warning: 3 | Critical: 14 | Inactive: 236 (no recent runs)
- Success rate (completed runs): 29% (24/82) — ↓ from 34% Jul 5 PM
- AR rate: 60% (49/82) — ↑ significantly from 22% Jul 5 PM (systemic pressure)

### P0 Issues (DO NOT RE-FILE)
- **Copilot BYOK stream_options (#43031 OPEN)**: SDK injects OpenAI-only `stream_options` into Anthropic calls → HTTP 400. PR #43127 open for fix. DO NOT RE-FILE.

### P1 Issues (DO NOT RE-FILE)
- **PR Sous Chef** (#43040, #43143): Missing `pr-processor` sub-agent. DO NOT RE-FILE.
- **Smoke CI** (#42398): EACCES mkdir. DO NOT RE-FILE.
- **CI Integration TestMCPGateway** (#42423): Failing. DO NOT RE-FILE.
- **Sub-Agent Model Resolution** (#42033, #43335): codex alpha 404. DO NOT RE-FILE.
- **Daily Safe Output Integrator** (#42333): Tool denial 5/5. DO NOT RE-FILE.
- **Daily BYOK Ollama** (#41827): api-proxy 503. DO NOT RE-FILE.
- **Go Logger Enhancement** (#42032): jq ARG_MAX. DO NOT RE-FILE.
- **Smoke Copilot Sub Agents** (#42824): 100% red. DO NOT RE-FILE.
- **AI Moderator** (#43352): cache_memory_miss + no-safe-outputs. DO NOT RE-FILE.
- **PR Code Quality Reviewer** (#aw_prcq_fix): regression. DO NOT RE-FILE.
- **Matt Pocock Skills Reviewer** (#43309): recurring failures. DO NOT RE-FILE.
- **Impeccable Skills Reviewer** (#43308): engine crash. DO NOT RE-FILE.
- **Metrics Collector** (#43292): Engine failure — data stale since Jan 2026. DO NOT RE-FILE.

### P2 Issues (DO NOT RE-FILE)
#41987,#41988,#42329,#42332,#42342,#42356,#42442,#42482,#42598,#42607,#42637,#42867,#42870,#42871,#42883,#42889,#42890,#42899,#42908,#42918,#42919,#42930,#42943,#42960,#43079,#43087,#43101,#43108,#43110,#43122,#43138,#43141,#43143,#43146,#43159,#43161,#43179,#43182,#43191,#43194,#43281,#43319,#43330,#43353,#43355,#43368,#43379

### New Observations (Jul 6)
- **AR rate spike (22% → 60%)**: Systemic — consistent with existing P0 BYOK and sandbox issues (no new root cause identified)
- **Content Moderation regression**: Was "Stable" Jul 5, now 1/5 success today — watching; matches broader AR spike pattern
- **Q workflow**: 0/20 success all AR — already tracked via #aw_quality_plateau
- **Design Decision Gate**: Showing AR again after Jul 5 recovery — intermittent, watching
- **Agentic Commands**: 45% success (7/20) — same pattern as Jul 5, no regression
- **CGO**: Mostly healthy today (3/4 success + 1 in_progress) — Jul 5 regression watch resolved
- **CWI**: 1/2 success — mixed but within normal range
- **Label Closed PRs / PR Description Updater**: Single AR each — isolated events, no issue warranted

### Confirmed Stable ✅
- Avenger | Daily Safe Outputs Git Simulator | Publish safe outputs Node image | Copilot | Copilot cloud agent | Test Quality Sentinel (17/20 success) | Smoke CI (partial) | Doc Build-Deploy (partial)

### Systemic Issues (unchanged)
1. **Copilot BYOK 400** (stream_options mismatch) → #43031 (P0)
2. **Sandbox seccomp restrictions** → #43110, #43101
3. **Smoke CI sandbox EACCES** → #42398
4. **Codex alpha 404** (9+ workflows) → #42033, #43335
5. **action_required systemic pressure** (~60% of today's runs — elevated)

### Actions Taken This Run
- No new issues filed (all patterns already tracked)
- AR spike noted — systemic root causes already tracked
- Shared memory updated

### Do Not Re-File (cumulative through Jul 6)
#41827,#41987,#41988,#42032,#42033,#42095,#42329,#42332,#42333,#42342,#42356,#42398,#42423,#42442,#42482,#42598,#42607,#42637,#42652,#42824,#42867,#42872,#42883,#42889,#42890,#42899,#42908,#42918,#42919,#42921,#42930,#42943,#42960,#43031,#43040,#43045,#43065,#43066,#43079,#43084,#43087,#43101,#43108,#43110,#43122,#43138,#43141,#43143,#43146,#43159,#43161,#43179,#43182,#43191,#43194,#43277,#43281,#43292,#43308,#43309,#43317,#43319,#43323,#43330,#43335,#43336,#43352,#43353,#43355,#43368,#43379,#aw_ci_parser_ctx,#aw_quality_plateau,#aw_whd_jul4,#aw_whd_jul5
