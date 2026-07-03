# Workflow Health — 2026-07-03T05:41Z

Score: 72/100 (↓3 from 75 Jul 1) | Run: §28640918061

## KEY FINDINGS

### Compilation Status
- **258/258 workflows have lock files (100% ✅)**. Compile-validate clean.

### P0 Issues
- **Copilot BYOK stream_options** (#43031 OPEN NEW): SDK injects OpenAI-only `stream_options` into Anthropic calls → HTTP 400, agent never executes. Affects BYOK workflows deterministically. DO NOT RE-FILE.

### P1 Issues
- **PR Sous Chef** (#43040 OPEN, #42652 closed): HTTP 400 recurring fresh failure today. DO NOT RE-FILE.
- **PR Code Quality Reviewer**: Fresh failure today run §28639714841, #42095 closed — may need new issue.
- **Smoke CI** (#42398 OPEN): EACCES mkdir /tmp/gh-aw. DO NOT RE-FILE.
- **CI Integration Test Regression** (#42423 OPEN): TestMCPGateway failing. DO NOT RE-FILE.
- **Sub-Agent Model Resolution Audit** (#42033 OPEN, #42921): codex alpha 404. DO NOT RE-FILE.
- **Daily Safe Output Integrator** (#42333 OPEN): Tool denial 5/5. DO NOT RE-FILE.
- **Daily BYOK Ollama** (#41827 OPEN): api-proxy 503. DO NOT RE-FILE.
- **Go Logger Enhancement** (#42032 OPEN): jq ARG_MAX. DO NOT RE-FILE.
- **Smoke Copilot Sub Agents** (#42824 OPEN): 100% red - no model. DO NOT RE-FILE.

### P2 Issues (New Today)
- **Daily yamllint Fixer** (#43108 OPEN): 3rd consecutive push rejection — allowed-files restriction. DO NOT RE-FILE.
- **Code Simplifier** (#43110 OPEN): Sandbox seccomp blocks binary exec (node/npm/go). DO NOT RE-FILE.
- **Smoke Copilot AOAI (apikey)** (#43101 OPEN): Missing required tool. DO NOT RE-FILE.
- **Smoke Antigravity** (#43087 OPEN): No safe outputs on PR run. DO NOT RE-FILE.
- **Impeccable Skills Reviewer** (#43079 OPEN): Engine failure (copilot engine terminated). DO NOT RE-FILE.

### Previously Tracked P2 (Still Open)
- #41987, #41988, #42329, #42332, #42342, #42356, #42442, #42482, #42598, #42607, #42637, #42867, #42870, #42871, #42883, #42889, #42890, #42899, #42908, #42918, #42919, #42930, #42943, #42960

### Confirmed Stable ✅
- Avenger (success today) | Auto-Close Parent Issues | Safe Output Health Monitor | CI

### Systemic Issues
1. **Copilot BYOK 400** (stream_options mismatch) → #43031 (P0)
2. **Sandbox seccomp restrictions** (blocks node/npm/go binary exec) → #43110, #43101
3. **Push restrictions (allowed-files)** → #43108, #41987
4. **Smoke CI sandbox EACCES** → #42398
5. **Model routing mismatch** (tier-unsupported, codex alpha 404) → #42033, #42921

### Run Summary (Jul 3 last 24h)
- Successes: 42 | action_required: 20 | Cancelled: 13 | Skipped: 17 | Failures: 3 | Running: 5
- Open [aw] issues: 30
- Dashboard created: #aw_whd_jul3

### Do Not Re-File
#41827,#41987,#41988,#42032,#42033,#42095(closed),#42329,#42332,#42333,#42342,#42356,#42398,#42423,#42442,#42482,#42598,#42607,#42637,#42652(closed),#42824,#42867,#42872,#42883,#42889,#42890,#42899,#42908,#42918,#42919,#42921,#42930,#42943,#42960,#43031,#43040,#43079,#43087,#43101,#43108,#43110
