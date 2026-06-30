# Workflow Health — 2026-06-30T05:52Z

Score: 78/100 (↓2 from 80 Jun 29) | Run: §28423332516

## KEY FINDINGS

### Status (June 30)
- **Compilation:** 257/257 workflows have lock files (100% ✅). Compile-validate clean.
- **CI Integration Test Regression (P1, NEW):** `TestMCPGatewayDockerCommandUsesRunnerIdentityAndSocketGroup` failing on main since 05:30. "Docker command should map host.docker.internal to host-gateway". CI was passing at 01:03 and 03:15 today — regression between 03:15–05:30. Filed NEW issue.
- **Sub-Agent Model Resolution Audit (P1, #42033 OPEN):** 100% red. Codex alpha 404. DO NOT RE-FILE.
- **PR Code Quality Reviewer (P1, #42095 OPEN):** Tier-unsupported model. DO NOT RE-FILE.
- **Daily BYOK Ollama Test (P1, #41827 OPEN):** api-proxy 503. DO NOT RE-FILE.
- **Go Logger Enhancement (P1, #42032 OPEN):** jq ARG_MAX. DO NOT RE-FILE.
- **Daily Safe Output Integrator (P1, #42333 OPEN):** Tool denial 5/5 again (Jun 29 19:40). DO NOT RE-FILE.
- **Changeset Generator (P2, #41987 OPEN):** workflows scope missing. DO NOT RE-FILE.
- **Smoke Copilot (P2, #41988 OPEN):** Missing `message` input. DO NOT RE-FILE.
- **PR Sous Chef (P2, #42370 OPEN):** 1 failure Jun 30 05:27. DO NOT RE-FILE.
- **Agentic Workflow Audit Agent (P2, #42356 OPEN):** Recurring failure. DO NOT RE-FILE.
- **Daily Team Evolution Insights (P2, #42342 OPEN):** Missing required tool. DO NOT RE-FILE.
- **Smoke CI hard-red (P2, #42398 OPEN):** EACCES mkdir /tmp/gh-aw. DO NOT RE-FILE.
- **Copilot Opt (P2, #42329 OPEN):** Tool denial limit. DO NOT RE-FILE.
- **AI Moderator (P2, #42332 OPEN):** Incomplete result. DO NOT RE-FILE.

### Recently Closed (Jun 29-30) ✅
- #42125 (Safe Output Integrator old), #42140 (Agentic Workflow Audit), #42105 (Formal Spec Verifier)
- #42124 (Code Metrics), #42128 (Team Evolution old), #42204 (Layout Spec Maintainer)
- #42234 (AI Moderator old), #42248 (GitHub MCP Structural Analysis)

### Confirmed Healthy (Jun 30) ✅
- **Avenger:** Running Jun 30 05:49 (no conclusion yet, expected STABLE)
- **Auto-Triage Issues:** STABLE ✅
- **Daily Semgrep Scan:** Running Jun 30 05:45

### Systemic Issues
1. **Codex alpha model 404** → #42033
2. **Tool denial 5/5** (Safe Output Integrator, Copilot Opt) → escalating pattern, #42333, #42329
3. **Missing tool declarations** (Team Evolution) → #42342

### Actions Taken (Jun 30)
- Created P1 issue for CI integration test regression
- Updated health dashboard #42186
- Updated shared-alerts.md and workflow-health-latest.md

## Do Not Re-File (Jun 30 state)
#41827, #41987, #41988, #42032, #42033, #42095, #42329, #42332, #42333, #42342, #42356, #42370, #42398
