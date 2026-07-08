# Workflow Health — 2026-07-08T05:50Z

Score: 63/100 (↓ from 65 Jul 7) | Run: §28919919125

## KEY FINDINGS

### Compilation Status
- **258/258 workflows have lock files (100% ✅)**. Compile-validate clean.

### Health Overview (Jul 8 early AM)
- Total runs sampled (today): 100 | Active workflows: 22
- Success: 36 (36%) | AR: 41 (41%) | In-progress: 9 (9%) | Skipped: 13 (13%)
- **Regression**: Success rate down from 42% (Jul 7) → 36% (Jul 8); AR rate up 39% → 41%

### P0 Issues (none new)

### P1 Issues
- **AI Moderator (#aw_ai_mod_jul8)**: NEW ISSUE FILED. Issue #43925 closed Jul 7 but 4/4 AR today. Codex engine failure + no safe outputs. DO NOT RE-FILE.
- **CGO (#38777)**: Escalated to 100% AR today (3/3). Commented on #38777 with escalation status. DO NOT RE-FILE.
- **Q (#43925 closed, persistent)**: 7/11 AR (63%) today. Unchanged. DO NOT RE-FILE.
- **Agentic Commands**: 7/12 AR (58%) today. Persistent pattern. DO NOT RE-FILE.
- **Smoke CI (#42398)**: 7/7 AR (100%) today. Persistent. DO NOT RE-FILE.
- **CJS**: 4/5 AR (80%) today on PRs — CI workflow. No prior issue.
- **CWI**: 2/4 AR (50%) today on PRs — CI workflow. No prior issue.
- **PR Sous Chef (#43143)**: Missing pr-processor. DO NOT RE-FILE.
- **Sub-Agent Model Resolution (#42033, #43335)**: codex alpha 404. DO NOT RE-FILE.
- **Daily Safe Output Integrator (#42333)**: Tool denial 5/5. DO NOT RE-FILE.
- **Daily BYOK Ollama (#41827, #43883)**: api-proxy 503. DO NOT RE-FILE.
- **Go Logger Enhancement (#42032)**: jq ARG_MAX. DO NOT RE-FILE.
- **Smoke Copilot Sub Agents (#42824)**: 100% red. DO NOT RE-FILE.
- **Matt Pocock Skills Reviewer (#43309, #43894)**: recurring. DO NOT RE-FILE.
- **Impeccable Skills Reviewer (#43308, #43895)**: engine crash. DO NOT RE-FILE.
- **Metrics Collector (#43292)**: Engine failure — data stale. DO NOT RE-FILE.
- **Daily yamllint Fixer (#43927)**: 3/10 success (30%). DO NOT RE-FILE.
- **Code Simplifier (#43930)**: 6/10 success (60%). DO NOT RE-FILE.

### P2 Issues
#41987,#41988,#42329,#42332,#42342,#42356,#42442,#42482,#42598,#42607,#42637,#42867,#42870,#42871,#42883,#42889,#42890,#42899,#42908,#42918,#42919,#42930,#42943,#42960,#43079,#43087,#43101,#43108,#43110,#43122,#43138,#43141,#43143,#43146,#43159,#43161,#43179,#43182,#43191,#43194,#43281,#43319,#43330,#43353,#43355,#43368,#43379

### Regressions Since Jul 7
- **AI Moderator**: issue #43925 closed prematurely — failures resumed immediately next day
- **CGO**: Escalated 1 AR (Jul 7 AM) → 2 fail+1 AR (Jul 7 PM) → 3/3 AR (Jul 8 AM)
- **Success rate**: 42% → 36% (slight decline)
- **Content Moderation**: 4/8 AR (50%) today — mixed (was stable earlier today)
- **CJS**: 4/5 AR (80%) — CI workflow, same-time pattern as CGO/CWI

### Confirmed Stable ✅
- Avenger | Daily Safe Outputs Git Simulator | Auto-Triage Issues | Agentic Maintenance | Daily Semgrep Scan | Deployment Incident Monitor | Auto-Close Parent Issues

### Systemic Issues (updated Jul 8)
1. **Codex engine 404** → AI Moderator, #43335 (persistent)
2. **CGO/CJS/CWI CI regression** → All 3 CI workflows went AR together after 05:14 today (suggests shared infra issue)
3. **Sandbox seccomp restrictions** → #43101, #43110 (persistent)
4. **Smoke CI sandbox EACCES** → #42398 (persistent)
5. **Q workflow persistent AR** → 63% today, quality gate PR #43527 not yet producing improvement
6. **Doc Build Deploy** → Pending Pages environment approval (#43102)

### Actions Taken This Run
- NEW ISSUE: AI Moderator recurring failure → #aw_ai_mod_jul8
- COMMENT: CGO escalation → #38777 (worsening: 100% AR today)
- COMMENT: Doc Build Deploy status update → #43102
- Shared memory updated

### Do Not Re-File (cumulative through Jul 8)
#41827,#41987,#41988,#42032,#42033,#42095,#42329,#42332,#42333,#42342,#42356,#42398,#42423,#42442,#42482,#42598,#42607,#42637,#42652,#42824,#42867,#42872,#42883,#42889,#42890,#42899,#42908,#42918,#42919,#42921,#42930,#42943,#42960,#43031,#43040,#43045,#43065,#43066,#43079,#43084,#43087,#43101,#43108,#43110,#43122,#43138,#43141,#43143,#43146,#43159,#43161,#43179,#43182,#43191,#43194,#43277,#43281,#43292,#43308,#43309,#43317,#43319,#43323,#43330,#43335,#43336,#43352,#43353,#43355,#43368,#43379,#43883,#43894,#43895,#43925,#43927,#43930,#44006,#44016,#44031,#44032,#44035,#aw_ai_mod_jul8,#aw_ci_parser_ctx,#aw_quality_plateau,#aw_whd_jul4,#aw_whd_jul5,#aw_whd_jul6,#aw_whd_jul7
