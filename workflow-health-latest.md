# Workflow Health — 2026-07-01T06:10Z

Score: 75/100 (↓3 from 78 Jun 30) | Run: §28496806312

## KEY FINDINGS

### Status (July 1)
- **Compilation:** 257/257 workflows have lock files (100% ✅). Compile-validate clean.
- **PR Sous Chef (P1, #42652 NEW):** HTTP 400 recurring AFTER fix #42444 was closed. Model routing gpt-5.5 issue still unresolved. DO NOT RE-FILE.
- **Daily Safe Output Integrator (P1, #42333 OPEN):** Tool denial 5/5 (4th recurrence). DO NOT RE-FILE.
- **Sub-Agent Model Resolution Audit (P1, #42033 OPEN):** Codex alpha 404. DO NOT RE-FILE.
- **PR Code Quality Reviewer (P1, #42095 OPEN):** Tier-unsupported model. DO NOT RE-FILE.
- **Daily BYOK Ollama Test (P1, #41827 OPEN):** api-proxy 503. DO NOT RE-FILE.
- **CI Integration Test Regression (P1, #42423 OPEN):** TestMCPGateway failing. DO NOT RE-FILE.
- **Go Logger Enhancement (P1, #42032 OPEN):** jq ARG_MAX. DO NOT RE-FILE.
- **Changeset Generator (P2, #41987 OPEN):** Push rejected. DO NOT RE-FILE.
- **Smoke Copilot (P2, #41988 OPEN):** Missing message input. DO NOT RE-FILE.
- **Agentic Workflow Audit Agent (P2, #42356 OPEN):** Recurring failure. DO NOT RE-FILE.
- **Daily Team Evolution Insights (P2, #42342 OPEN):** Missing required tool. DO NOT RE-FILE.
- **Smoke CI (P2, #42398 OPEN):** EACCES mkdir /tmp/gh-aw. DO NOT RE-FILE.
- **Copilot Opt (P2, #42329 OPEN):** Tool denial limit. DO NOT RE-FILE.
- **AI Moderator (P2, #42332 OPEN):** Incomplete result. DO NOT RE-FILE.
- **Claude Code User Docs Review (P2, #42482 OPEN):** cache-memory miss. DO NOT RE-FILE.
- **Daily Hippo Learn (P2, #42442 OPEN):** hippo MCP tool unavailable. DO NOT RE-FILE.
- **Daily yamllint Fixer (P2, #42637 OPEN):** Code push failed. DO NOT RE-FILE.
- **claude-sonnet-5 retired (P2, #42598 OPEN):** Affects daily-model-inventory. DO NOT RE-FILE.
- **Auto-Triage Issues (P2, #42607 OPEN):** PI engine transient failure (1/10, likely self-recovered). DO NOT RE-FILE.
- **Daily Credit Limit Test (P2, #42610 OPEN):** Credits exceeded. DO NOT RE-FILE.

### New Today (Jul 1)
- **PR Sous Chef recurrence:** #42652 auto-filed. Previous fix #42444 closed Jun 30 22:50 failed to hold.
- **claude-sonnet-5 retired:** #42598 auto-filed. Only affects daily-model-inventory.md.
- **Daily yamllint Fixer push failure:** #42637 auto-filed.
- **Auto-Triage transient:** #42607 auto-filed (single failure, likely recovered).
- **Credit limit exceeded:** #42610 auto-filed.
- **Dashboard created:** #42656

### Confirmed Healthy (Jul 1) ✅
- **Avenger:** STABLE
- **Issue Monster:** 100%
- **Daily Semgrep Scan:** Running

### Systemic Issues
1. **Model routing mismatch** (gpt-5.5, claude-sonnet-5 retiring) → #42652, #42598
2. **Tool denial guardrails 5/5** → #42333, #42329
3. **Missing tool declarations** → #42342, #42442
4. **Codex alpha 404** → #42033
5. **Code push rejected** → #42637, #41987

### Actions Taken (Jul 1 06:10Z)
- No new issues filed (all captured by auto-filing)
- Created health dashboard issue #42656
- Updated shared-alerts.md and workflow-health-latest.md

## Do Not Re-File (Jul 1 state)
#41827, #41987, #41988, #42032, #42033, #42095, #42329, #42332, #42333, #42342, #42356, #42398, #42421, #42423, #42442, #42444, #42482, #42598, #42607, #42610, #42637, #42652, #42656
