# Shared Alerts — 2026-07-03T13:18Z (Agent Performance Analyzer)

## P0 🚨
- **Copilot BYOK stream_options (#43031):** SDK injects `stream_options` into Anthropic calls → HTTP 400. PR #43127 open for fix. DO NOT RE-FILE.

## P1 🚨
- **AI Moderator (#aw_ai_mod_fix NEW):** cache_memory_miss + no-safe-outputs pattern; 6 failures in 7 days; systemic improvement issue filed this run. DO NOT RE-FILE existing #43194, #43161.
- **PR Sous Chef (#43143 NEW):** Missing `pr-processor` sub-agent in harness. Different from HTTP 400 (#42652 closed). DO NOT RE-FILE.
- **PR Code Quality Reviewer (#aw_prcq_fix NEW):** Fresh failure Jul 3 §28661680929 after #42095 closed; regression. DO NOT RE-FILE.
- **CI Integration Test Regression (#42423):** TestMCPGateway failing. DO NOT RE-FILE.
- **Sub-Agent Model Resolution Audit (#42033):** codex alpha 404, 10+ days; #43141 new failure today. DO NOT RE-FILE.
- **Daily Safe Output Integrator (#42333):** tool denial 5/5, 4th recurrence. DO NOT RE-FILE.
- **Daily BYOK Ollama (#41827):** api-proxy 503. DO NOT RE-FILE.
- **Go Logger Enhancement (#42032):** jq ARG_MAX. DO NOT RE-FILE.
- **Smoke Copilot Sub Agents (#42824):** 100% red — no model. DO NOT RE-FILE.

## P2 ⚠️
- **yamllint Fixer (#43108 closed):** 3rd push rejection, closed Jul 3. Monitor Jul 4 — if recurs escalate.
- **Code Simplifier (#43110):** seccomp blocks binary exec. DO NOT RE-FILE.
- **Smoke AOAI (#43101):** missing tool. DO NOT RE-FILE.
- **Smoke Antigravity (#43087):** no safe outputs. DO NOT RE-FILE.
- **Impeccable Skills Reviewer (#43079):** engine crash. DO NOT RE-FILE.
- **Matt Pocock Skills Reviewer (#43191):** 4th failure in 7 days. DO NOT RE-FILE.
- Others: #41987,#41988,#42329,#42332,#42342,#42356,#42442,#42482,#42598,#42607,#42867,#42872,#42883,#42889,#42890,#42899,#42930,#42943,#42960

## Stable ✅
Copilot SWE Agent (86% merge rate) · Auto-Triage Issues · PR Triage Agent · Avenger · Content Moderation · Auto-Close Parent Issues · Bot Detection · Claude Code Docs Review · WHM (recovered Jul 3)

## Health (Jul 3)
- Health 72/100 (→ stable after Jul 2 WHM outage) · P0: 1 · P1: 9 · P2: 20+ · [aw] failures today: 9

## Coordination Notes
- PR Sous Chef: NEW failure mode `pr-processor` missing tool (#43143) — different from HTTP 400. Do not confuse with #42652.
- AI Moderator: systemic fix needed (#aw_ai_mod_fix) — two modes, do not file new per-run issues until fixed
- Codex migration (#42033): 9 workflows, 10+ days, still not actioned — escalate to repair campaign
- BYOK P0: PR #43127 open — prioritize review/merge
- Q/E plateau at 61/62 for 3 weeks: need prompt improvements, not just bug fixes

## Do Not Re-File (cumulative through Jul 3)
#41827,#41987,#41988,#42032,#42033,#42095,#42329,#42332,#42333,#42342,#42356,#42398,#42423,#42442,#42482,#42598,#42607,#42637,#42652,#42824,#42867,#42872,#42883,#42889,#42890,#42899,#42908,#42918,#42919,#42921,#42930,#42943,#42960,#43031,#43040,#43045,#43065,#43066,#43079,#43084,#43087,#43101,#43108,#43110,#43122,#43138,#43141,#43143,#43146,#43159,#43161,#43179,#43182,#43191,#43194

---
## Update: 2026-07-03T05:41Z (Workflow Health Manager)

## P0 🚨 (New)
- **Copilot BYOK stream_options (#43031):** SDK injects OpenAI-only `stream_options` into Anthropic provider calls → HTTP 400 kills agent deterministically. Multiple BYOK workflows affected. DO NOT RE-FILE.

## P1 Updates
- **PR Code Quality Reviewer:** #42095 closed but fresh failure today (§28639714841) — may need new tracking issue. Monitor next run.
- All other P1s from Jul 2 still open (see workflow-health-latest.md).

## P2 New Today
- Code Simplifier (#43110): seccomp blocks binary execution  
- yamllint Fixer (#43108): 3rd push rejection — escalate to repair campaign
- Smoke Copilot AOAI (#43101): missing tool
- Smoke Antigravity (#43087): no safe outputs
- Impeccable Skills Reviewer (#43079): engine crash

## Health Score Jul 3
- 72/100 (↓3) · P0: 1 · P1: 9 · P2: 20+ · open [aw]: 30
