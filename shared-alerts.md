# Shared Alerts — 2026-06-30T13:20Z (updated by Agent Performance Analyzer)

## P1 🚨
- **CI Integration Test Regression (NEW, Jun 30)**: `TestMCPGatewayDockerCommandUsesRunnerIdentityAndSocketGroup` failing on main. #42423 OPEN. DO NOT RE-FILE.
- **PR Sous Chef P1 ESCALATION (NEW, Jun 30)**: 5/5 failures. `gpt-5.5` routed via `/chat/completions` → 400. Fix: #42421 in-flight. Issue: #42444 OPEN. DO NOT RE-FILE.
- **Sub-Agent Model Resolution Audit (#42033 OPEN)**: 100% red since Jun 24. Codex alpha 404. DO NOT RE-FILE.
- **PR Code Quality Reviewer (#42095 OPEN)**: Tier-unsupported model → SDK 400. DO NOT RE-FILE.
- **Daily Safe Output Integrator (#42333 OPEN)**: Tool denial 5/5 again (4th instance). DO NOT RE-FILE.
- **Daily BYOK Ollama Test (#41827 OPEN)**: api-proxy 503. DO NOT RE-FILE.
- **Go Logger Enhancement (#42032 OPEN)**: jq ARG_MAX. DO NOT RE-FILE.

## P2 ⚠️ (Monitor)
- **Claude Code User Documentation Review (#42482 OPEN, NEW)**: cache-memory miss misconfiguration. DO NOT RE-FILE.
- **Daily Hippo Learn (#42442 OPEN, NEW)**: hippo MCP tool unavailable (0 tools exposed). DO NOT RE-FILE.
- **Changeset Generator (#41987 OPEN)**: Push rejected. DO NOT RE-FILE.
- **Smoke Copilot (#41988 OPEN)**: Missing `message` input. DO NOT RE-FILE.
- **Agentic Workflow Audit Agent (#42356 OPEN)**: Recurring failure. DO NOT RE-FILE.
- **Daily Team Evolution Insights (#42342 OPEN)**: Missing required tool. DO NOT RE-FILE.
- **Smoke CI hard-red (#42398 OPEN)**: EACCES mkdir /tmp/gh-aw. DO NOT RE-FILE.
- **Copilot Opt (#42329 OPEN)**: Tool denial limit. DO NOT RE-FILE.
- **AI Moderator (#42332 OPEN)**: Incomplete result. DO NOT RE-FILE.

## Confirmed Stable ✅
- **Compilation**: 257/257 ✅
- **Issue Monster**: 100% (5/5)
- **Auto-Triage Issues**: 100% (5/5) ✅
- **Avenger**: 100% ✅

## Health Scores (Jun 30 13:20Z)
- Health Score: 78/100 (↓2)
- P1 issues: 6 (CI regression + PR Sous Chef NEW + 4 carry-forward)
- P2 issues: 10

## Systemic Issues
1. **Model routing mismatch**: gpt-5.5 → /chat/completions instead of /responses. #42444, fix #42421
2. **Codex alpha model 404**: tracked #42033
3. **Tool denial guardrails 5/5**: escalating — Safe Output Integrator, Copilot Opt
4. **Missing tool declarations**: Team Evolution #42342, Hippo Learn #42442
5. **Harness retry waste**: burns all 4 retries on non-retryable 400s (documented in #42444)

## Coordination Notes (for Campaign Manager + Workflow Health Manager)
- PR Sous Chef escalated to P1 Jun 30 morning — triage campaigns dependent on PR Sous Chef outputs are paused
- Harness retry-waste pattern has cost implications; consider extracting dedicated improvement issue from #42444
- Missing tool pattern (2 agents) suggests MCP tool registration audit needed

## Do Not Re-File (Jun 30 13:20Z state)
#41827, #41987, #41988, #42032, #42033, #42095, #42329, #42332, #42333, #42342, #42356, #42370, #42398, #42421, #42423, #42444, #42471, #42482, #42442
