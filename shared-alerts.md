# Shared Alerts — 2026-07-01T06:10Z (updated by Workflow Health Manager)

## P1 🚨
- **PR Sous Chef (P1 RECURRING, Jul 1):** HTTP 400 post-fix recurrence. #42652 OPEN. Fix #42444 closed Jun 30 22:50 but problem persists. DO NOT RE-FILE.
- **CI Integration Test Regression (#42423 OPEN):** `TestMCPGatewayDockerCommandUsesRunnerIdentityAndSocketGroup` failing on main. DO NOT RE-FILE.
- **Sub-Agent Model Resolution Audit (#42033 OPEN):** 100% red since Jun 24. Codex alpha 404. DO NOT RE-FILE.
- **PR Code Quality Reviewer (#42095 OPEN):** Tier-unsupported model → SDK 400. DO NOT RE-FILE.
- **Daily Safe Output Integrator (#42333 OPEN):** Tool denial 5/5 (4th recurrence). DO NOT RE-FILE.
- **Daily BYOK Ollama Test (#41827 OPEN):** api-proxy 503. DO NOT RE-FILE.
- **Go Logger Enhancement (#42032 OPEN):** jq ARG_MAX. DO NOT RE-FILE.

## P2 ⚠️ (Monitor)
- **Daily yamllint Fixer (#42637 OPEN, NEW Jul 1):** Code push failed (create_pull_request rejected). DO NOT RE-FILE.
- **claude-sonnet-5 retired (#42598 OPEN, NEW Jul 1):** Only affects daily-model-inventory.md. DO NOT RE-FILE.
- **Auto-Triage Issues (#42607 OPEN):** PI engine transient (1/10 runs, likely self-recovered). DO NOT RE-FILE.
- **Daily Credit Limit Test (#42610 OPEN):** Credits exceeded. DO NOT RE-FILE.
- **Claude Code User Documentation Review (#42482 OPEN):** cache-memory miss misconfiguration. DO NOT RE-FILE.
- **Daily Hippo Learn (#42442 OPEN):** hippo MCP tool unavailable (0 tools exposed). DO NOT RE-FILE.
- **Changeset Generator (#41987 OPEN):** Push rejected. DO NOT RE-FILE.
- **Smoke Copilot (#41988 OPEN):** Missing `message` input. DO NOT RE-FILE.
- **Agentic Workflow Audit Agent (#42356 OPEN):** Recurring failure. DO NOT RE-FILE.
- **Daily Team Evolution Insights (#42342 OPEN):** Missing required tool. DO NOT RE-FILE.
- **Smoke CI (#42398 OPEN):** EACCES mkdir /tmp/gh-aw. DO NOT RE-FILE.
- **Copilot Opt (#42329 OPEN):** Tool denial limit. DO NOT RE-FILE.
- **AI Moderator (#42332 OPEN):** Incomplete result. DO NOT RE-FILE.

## Confirmed Stable ✅
- **Compilation**: 257/257 ✅
- **Avenger**: 100% ✅
- **Issue Monster**: 100% ✅
- **Auto-Triage Issues**: 9/10 (1 transient PI engine crash; likely recovered)

## Health Scores (Jul 1 06:10Z)
- Health Score: 75/100 (↓3)
- P1 issues: 7
- P2 issues: 13
- Dashboard: #42656

## Systemic Issues
1. **Model routing mismatch**: gpt-5.5 → /chat/completions; claude-sonnet-5 retiring. #42652, #42598
2. **Tool denial guardrails 5/5**: escalating — Safe Output Integrator, Copilot Opt. #42333, #42329
3. **Missing tool declarations**: Team Evolution, Hippo Learn. #42342, #42442
4. **Codex alpha 404**: tracked #42033
5. **Code push rejected**: yamllint, changeset. #42637, #41987

## Coordination Notes (for Campaign Manager + Agent Performance Analyzer)
- PR Sous Chef recurrence post-fix (#42652): triage campaigns dependent on PR Sous Chef outputs remain paused
- Auto-Triage transient (#42607): only 1 failure in 10 runs, likely self-recovered; campaigns can proceed with caution
- Model deprecation pattern growing: codex alpha, gpt-5.5, claude-sonnet-5 — recommend systematic model version audit
- Harness retry-waste pattern (#42444 context): burns retries on non-retryable 400s
- yamllint Fixer (#42637): push rejection may indicate permissions issue — check GITHUB_TOKEN scope

## Do Not Re-File (Jul 1 06:10Z state)
#41827, #41987, #41988, #42032, #42033, #42095, #42329, #42332, #42333, #42342, #42356, #42398, #42421, #42423, #42442, #42444, #42482, #42598, #42607, #42610, #42637, #42652, #42656
