# Shared Alerts — 2026-07-08T05:50Z (Workflow Health Manager)

## P1 🚨
- **AI Moderator (#aw_ai_mod_jul8 NEW):** Issue #43925 closed Jul 7 but 4/4 AR today. Codex engine failure recurring. DO NOT RE-FILE.
- **CGO (#38777 escalated):** 100% AR today (was worsening since Jul 7 PM). WHM commented on #38777. DO NOT RE-FILE.
- **Q workflow**: 7/11 AR (63%) today. Quality gate PR #43527 not yet impacting. DO NOT RE-FILE.
- **Agentic Commands**: 7/12 AR (58%) — persistent. DO NOT RE-FILE.
- **PR Sous Chef (#43143):** Missing pr-processor. DO NOT RE-FILE.
- **Smoke CI (#42398, #43908):** EACCES mkdir. DO NOT RE-FILE.
- **CI Integration TestMCPGateway (#42423):** Failing. DO NOT RE-FILE.
- **Sub-Agent Model Resolution (#42033, #43335):** codex alpha 404. DO NOT RE-FILE.
- **Daily Safe Output Integrator (#42333):** Tool denial 5/5. DO NOT RE-FILE.
- **Daily BYOK Ollama (#41827, #43883):** api-proxy 503. DO NOT RE-FILE.
- **Smoke Copilot Sub Agents (#42824):** 100% red. DO NOT RE-FILE.
- **Matt Pocock Skills Reviewer (#43309, #43894):** recurring. DO NOT RE-FILE.
- **Impeccable Skills Reviewer (#43308, #43895):** engine crash pattern. DO NOT RE-FILE.
- **Metrics Collector (#43292):** Engine failure — data stale since Jan 2026. DO NOT RE-FILE.
- **Daily yamllint Fixer (#43927):** 3/10 success (30%). DO NOT RE-FILE.
- **Code Simplifier (#43930):** 6/10 success (60%). DO NOT RE-FILE.

## P2 ⚠️ Watch (Jul 8)
- **CJS**: 4/5 AR (80%) today on CI PRs — correlated with CGO/CWI pattern. Watch.
- **CWI**: 2/4 AR (50%) today on CI PRs — correlated. Watch.
- **Content Moderation**: 4/8 AR (50%) today — mixed batches. Watch for stabilization.
- **Smoke AOAI (#44031, #44032, #44035)**: incomplete/missing-tool errors. DO NOT RE-FILE.

## Systemic Issues (Jul 8)
1. **Codex engine 404** (#43335) — AI Moderator + others affected (persistent)
2. **CGO/CJS/CWI CI regression** — ALL 3 CI workflows went AR together after 05:14Z today. Possibly related to commit c94f432 (required-labels/required-title-prefix parser fix). Urgent investigation needed.
3. **Sandbox seccomp** → #43101, #43110 (persistent)
4. **Smoke CI sandbox EACCES** → #42398 (persistent)
5. **Q workflow persistent AR** — 63% today, quality gate not yet effective

## P2 ⚠️ (Existing — DO NOT RE-FILE)
#41987,#41988,#42329,#42332,#42342,#42356,#42442,#42482,#42598,#42607,#42637,#42867,#42872,#42883,#42889,#42890,#42899,#42930,#42943,#42960,#43079,#43087,#43101,#43108,#43110,#43122,#43138,#43141,#43143,#43146,#43159,#43161,#43179,#43182,#43191,#43194,#43281,#43319,#43330,#43353,#43355,#43368,#43379,#43281,#43317,#43323

## Actions Taken Jul 8
- Created issue #aw_ai_mod_jul8 (AI Moderator recurring failure)
- Commented on #38777 (CGO escalation — now 100% AR)
- Commented on #43102 (Doc Build Deploy status)

## Do Not Re-File (cumulative through Jul 8)
#41827,#41987,#41988,#42032,#42033,#42095,#42329,#42332,#42333,#42342,#42356,#42398,#42423,#42442,#42482,#42598,#42607,#42637,#42652,#42824,#42867,#42872,#42883,#42889,#42890,#42899,#42908,#42918,#42919,#42921,#42930,#42943,#42960,#43031,#43040,#43045,#43065,#43066,#43079,#43084,#43087,#43101,#43108,#43110,#43122,#43138,#43141,#43143,#43146,#43159,#43161,#43179,#43182,#43191,#43194,#43277,#43281,#43292,#43308,#43309,#43317,#43319,#43323,#43330,#43335,#43336,#43352,#43353,#43355,#43368,#43379,#43883,#43894,#43895,#43925,#43927,#43930,#44006,#44016,#44031,#44032,#44035,#aw_ai_mod_jul8,#aw_ci_parser_ctx,#aw_quality_plateau,#aw_whd_jul4,#aw_whd_jul5,#aw_whd_jul6,#aw_whd_jul7
