# Workflow Health — 2026-07-07T05:50Z

Score: 65/100 (↑ from 62 Jul 6) | Run: §28844636702

## KEY FINDINGS

### Compilation Status
- **258/258 workflows have lock files (100% ✅)**. Compile-validate clean.

### Health Overview (Jul 7 early AM)
- Total runs sampled (today): 100 | Active workflows: 36 | Inactive: ~222
- Success: 42 (42%) | AR: 39 (39%) | Failed: 4 (4%) | Skipped: 13 (13%)
- **Improvement**: Success rate up from 29% (Jul 6) → 42% (Jul 7); AR rate down from 60% → 39%

### P0 Issues
- **Copilot BYOK stream_options (#43031)**: PR #43127 MERGED Jul 4. AR rate measurably down (60% → 39%). Issue still OPEN — commented with closure recommendation.
  
### P1 Issues (DO NOT RE-FILE)
- **AI Moderator (#43925)**: New issue filed today (old #43352 closed). Still 1/15 success despite PR #43525 merge. AR pattern ongoing.
- **PR Sous Chef (#43143)**: Missing `pr-processor` sub-agent. DO NOT RE-FILE.
- **Smoke CI (#42398, #43908)**: EACCES mkdir. DO NOT RE-FILE.
- **CI Integration TestMCPGateway (#42423)**: Failing. DO NOT RE-FILE.
- **Sub-Agent Model Resolution (#42033, #43335)**: codex alpha 404. DO NOT RE-FILE.
- **Daily Safe Output Integrator (#42333)**: Tool denial 5/5. DO NOT RE-FILE.
- **Daily BYOK Ollama (#41827, #43883)**: api-proxy 503. DO NOT RE-FILE.
- **Go Logger Enhancement (#42032)**: jq ARG_MAX. DO NOT RE-FILE.
- **Smoke Copilot Sub Agents (#42824)**: 100% red. DO NOT RE-FILE.
- **Matt Pocock Skills Reviewer (#43309, #43894)**: recurring. DO NOT RE-FILE.
- **Impeccable Skills Reviewer (#43308, #43895)**: engine crash pattern. DO NOT RE-FILE.
- **Metrics Collector (#43292)**: Engine failure. DO NOT RE-FILE.
- **Daily yamllint Fixer (#43927)**: Just filed today (3/10 success, 30%). DO NOT RE-FILE.
- **Code Simplifier (#43930)**: Just filed today (6/10 success, 60%, alternating). DO NOT RE-FILE.

### P2 Issues (DO NOT RE-FILE)
#41987,#41988,#42329,#42332,#42342,#42356,#42442,#42482,#42598,#42607,#42637,#42867,#42870,#42871,#42883,#42889,#42890,#42899,#42908,#42918,#42919,#42930,#42943,#42960,#43079,#43087,#43101,#43108,#43110,#43122,#43138,#43141,#43143,#43146,#43159,#43161,#43179,#43182,#43191,#43194,#43281,#43319,#43330,#43353,#43355,#43368,#43379

### Improvements Since Jul 6
- **AR spike recovering**: 60% → 39% (BYOK fix #43127 taking effect)
- **PRCQ workflow**: Normal operation restored after PR #43730 merge (monitored stable)
- **CGO**: Healthy (1 AR today, 1 run)
- **Content Moderation**: 7/15 success today — mixed (AR batches + success batches suggesting intermittent engine issue)

### Confirmed Stable ✅
- Avenger | Daily Safe Outputs Git Simulator | Copilot cloud agent | Auto-Triage Issues | Agentic Maintenance | Extension Upgrade Test | GPL Dependency Cleaner | Go Logger Enhancement | Issue Monster | LintMonster | Refactoring Cadence | Safe Output Health Monitor | Sergo | jsweep | Designer Drift Audit | Documentation Noob Tester | Daily AstroStyleLite

### Systemic Issues (updated)
1. **Copilot BYOK stream_options** (#43031) — PR #43127 merged, AR rate improving ↑ (was P0, now monitoring)
2. **Sandbox seccomp restrictions** → #43101, #43110 (persistent)
3. **Smoke CI sandbox EACCES** → #42398 (persistent)
4. **Codex alpha 404** (12+ workflows) → #42033, #43335 (persistent)
5. **AI Moderator cache_memory_miss** → #43925 (new issue, ongoing)

### Actions Taken This Run
- Added status update comment to #43031 (BYOK P0) noting PR #43127 merged, recommending closure
- No new issues filed (all patterns tracked: yamllint #43927, code-simplifier #43930, AI moderator #43925 all auto-filed today)
- Shared memory updated

### Do Not Re-File (cumulative through Jul 7)
#41827,#41987,#41988,#42032,#42033,#42095,#42329,#42332,#42333,#42342,#42356,#42398,#42423,#42442,#42482,#42598,#42607,#42637,#42652,#42824,#42867,#42872,#42883,#42889,#42890,#42899,#42908,#42918,#42919,#42921,#42930,#42943,#42960,#43031,#43040,#43045,#43065,#43066,#43079,#43084,#43087,#43101,#43108,#43110,#43122,#43138,#43141,#43143,#43146,#43159,#43161,#43179,#43182,#43191,#43194,#43277,#43281,#43292,#43308,#43309,#43317,#43319,#43323,#43330,#43335,#43336,#43352,#43353,#43355,#43368,#43379,#43883,#43894,#43895,#43925,#43927,#43930,#aw_ci_parser_ctx,#aw_quality_plateau,#aw_whd_jul4,#aw_whd_jul5,#aw_whd_jul6
