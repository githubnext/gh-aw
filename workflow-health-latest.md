# Workflow Health — 2026-07-04T05:41Z

Score: 69/100 (↓3 from 72 Jul 3) | Run: §28696525534

## KEY FINDINGS

### Compilation Status
- **258/258 workflows have lock files (100% ✅)**. Compile-validate clean.

### P0 Issues
- **Copilot BYOK stream_options (#43031 OPEN)**: SDK injects OpenAI-only `stream_options` into Anthropic calls → HTTP 400. PR #43127 open for fix. DO NOT RE-FILE.

### P1 Issues (New Today)
- **CI Integration Parser context import (#aw_ci_parser_ctx NEW)**: `remote_fetch_integration_test.go:6` unused `context` import breaks `Integration: Parser Remote Fetch & Cache` + `Integration Unauthenticated Add` on main. Previous PR #43306 closed without merge. #43316 fixed lint/CJS but missed this. DO NOT RE-FILE.

### P1 Issues (Previously Tracked — DO NOT RE-FILE)
- **PR Sous Chef** (#43040): HTTP 400 / pr-processor missing tool (#43143). DO NOT RE-FILE.
- **Smoke CI** (#42398): EACCES mkdir /tmp/gh-aw. DO NOT RE-FILE.
- **CI Integration TestMCPGateway** (#42423): Failing. DO NOT RE-FILE.
- **Sub-Agent Model Resolution** (#42033): codex alpha 404. DO NOT RE-FILE.
- **Daily Safe Output Integrator** (#42333): Tool denial 5/5. DO NOT RE-FILE.
- **Daily BYOK Ollama** (#41827): api-proxy 503. DO NOT RE-FILE.
- **Go Logger Enhancement** (#42032): jq ARG_MAX. DO NOT RE-FILE.
- **Smoke Copilot Sub Agents** (#42824): 100% red - no model. DO NOT RE-FILE.
- **AI Moderator** (#aw_ai_mod_fix): cache_memory_miss + no-safe-outputs. DO NOT RE-FILE.
- **PR Code Quality Reviewer** (#aw_prcq_fix): fresh failure regression. DO NOT RE-FILE.

### P2 Issues (Previously Tracked — DO NOT RE-FILE)
#41987, #41988, #42329, #42332, #42342, #42356, #42442, #42482, #42598, #42607, #42637, #42867, #42870, #42871, #42883, #42889, #42890, #42899, #42908, #42918, #42919, #42930, #42943, #42960, #43079, #43087, #43101, #43108, #43110, #43122, #43138, #43141, #43143, #43146, #43159, #43161, #43179, #43182, #43191, #43194

### Recent Fixes ✅
- **#43316 merged**: Fixed `log_parser_bootstrap.test.cjs` async/await + `arc_dind_artifacts.go` modernize lint
- **Action**: CGO main lint failures resolved (pending CI re-run)

### Confirmed Stable ✅
- Avenger | Auto-Close Parent Issues | Safe Output Health Monitor | CI (partial) | Bot Detection | Content Moderation | Auto-Triage Issues

### Systemic Issues
1. **Copilot BYOK 400** (stream_options mismatch) → #43031 (P0)
2. **Sandbox seccomp restrictions** (blocks node/npm/go binary exec) → #43110, #43101
3. **CI integration build breakage** (context import) → #aw_ci_parser_ctx (P1, NEW)
4. **Smoke CI sandbox EACCES** → #42398
5. **Codex alpha 404** (9+ workflows, 10+ days) → #42033

### Run Summary (Jul 4 first ~2h)
- Successes: 39 | action_required: 22 | Failures: 12 | Skipped: 16 | Running/Cancelled: 11
- Dashboard created: #aw_whd_jul4
- New P1 issues created: 1 (#aw_ci_parser_ctx)

### Do Not Re-File
#41827,#41987,#41988,#42032,#42033,#42095,#42329,#42332,#42333,#42342,#42356,#42398,#42423,#42442,#42482,#42598,#42607,#42637,#42652,#42824,#42867,#42872,#42883,#42889,#42890,#42899,#42908,#42918,#42919,#42921,#42930,#42943,#42960,#43031,#43040,#43045,#43065,#43066,#43079,#43084,#43087,#43101,#43108,#43110,#43122,#43138,#43141,#43143,#43146,#43159,#43161,#43179,#43182,#43191,#43194,#43306,#aw_ci_parser_ctx,#aw_whd_jul4
