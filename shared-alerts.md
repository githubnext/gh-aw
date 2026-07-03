# Shared Alerts — 2026-07-02T13:16Z (Agent Performance Analyzer)

## P1 🚨
- **PR Sous Chef (#42652):** HTTP 400 recurring; pi-switch (#42730) validating — check Jul 3. DO NOT RE-FILE.
- **Workflow Health Manager (#42908 NEW):** repo-memory push failure Jul 2 05:55Z — health blind spot today. DO NOT RE-FILE.
- **CI Integration Test Regression (#42423):** TestMCPGateway failing. DO NOT RE-FILE.
- **Sub-Agent Model Resolution Audit (#42033):** codex alpha 404, 9+ days; recurrence #42921 Jul 2. DO NOT RE-FILE.
- **PR Code Quality Reviewer (#42095):** tier-unsupported model. DO NOT RE-FILE.
- **Daily Safe Output Integrator (#42333):** tool denial 5/5, 4th recurrence. DO NOT RE-FILE.
- **Daily BYOK Ollama (#41827):** api-proxy 503. DO NOT RE-FILE.
- **Go Logger Enhancement (#42032):** jq ARG_MAX. DO NOT RE-FILE.
- **Smoke Copilot Sub Agents (#42824):** 100% red — no model. DO NOT RE-FILE.

## P2 ⚠️ (Selected new + persisting)
- **yamllint Fixer (#42890, #42637):** 3rd consecutive push-rejection — ESCALATE to repair campaign. DO NOT RE-FILE.
- **Credit Limit recurring (#42872, #42610):** 6 AIC vs 1 threshold, 3rd occurrence — reschedule test earlier. DO NOT RE-FILE.
- **Daily Max Ai Credits Test (#42943):** copilot engine crash (rimraf/Node.js). DO NOT RE-FILE.
- **GitHub Remote MCP Auth Test (#42918):** new engine failure. DO NOT RE-FILE.
- **Multi-Device Docs Tester (#42919):** new failure. DO NOT RE-FILE.
- **AI Moderator (#42889, #42332):** no safe outputs. DO NOT RE-FILE.
- **Daily Compiler Quality Check (#42883):** incomplete result. DO NOT RE-FILE.
- **Smoke CI (#42899, #42398):** EACCES mkdir. DO NOT RE-FILE.
- **Smoke Antigravity (#42960):** no safe outputs. DO NOT RE-FILE.
- Others: #42482, #42442, #41987, #41988, #42342, #42356, #42329, #42598, #42607, #42637, #42867, #42870, #42871, #42930

## Stable ✅
Copilot SWE Agent (83% merge rate) · Issue Monster · PR Triage Agent · Auto-Triage Issues · Avenger · Team Status · AIC Consumption · Agentic Token Audit

## Health (Jul 2)
- Health ~72/100 (↓3) · P1s: 9 · P2s: 20+ · [aw] failures: 15

## Coordination Notes
- PR Sous Chef pi-switch (#42730): VALIDATE next run — do not close #42652 yet
- WHM down Jul 2: Campaign Manager has stale health data (last WHM: Jul 1 06:10Z)
- Credit limit: run Daily Credit Limit Test BEFORE high-consumption workflows
- Agentic Commands: 77% action_required rate (11/14) — approval-gated, expected behavior
- yamllint Fixer: 3 failures = assign repair or campaign priority

## Do Not Re-File (Jul 2)
#41827,#41987,#41988,#42032,#42033,#42095,#42329,#42332,#42333,#42342,#42356,#42398,#42421,#42423,#42442,#42444,#42482,#42598,#42607,#42610,#42637,#42652,#42656,#42730,#42765,#42766,#42824,#42867,#42872,#42883,#42889,#42890,#42899,#42908,#42918,#42919,#42921,#42930,#42943,#42960,#42966,#42970,#42971

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

## Do Not Re-File (cumulative through Jul 3)
#41827,#41987,#41988,#42032,#42033,#42329,#42332,#42333,#42342,#42356,#42398,#42423,#42442,#42482,#42598,#42607,#42637,#42824,#42867,#42872,#42883,#42889,#42890,#42899,#42908,#42918,#42919,#42921,#42930,#42943,#42960,#43031,#43040,#43079,#43087,#43101,#43108,#43110
