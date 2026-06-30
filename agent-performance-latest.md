# Agent Performance — Latest Run

**Timestamp:** 2026-06-30T13:20Z | **Run:** [§28447234062](https://github.com/github/gh-aw/actions/runs/28447234062)

## Summary: 62/100 Quality (↓1) | 63/100 Effectiveness (↓1) | 78/100 Health (→ stable)

## Top Performers
1. Copilot SWE Agent (Q:92, E:91) — 80% merge rate (61/76 settled), highest-volume contributor
2. Issue Monster (Q:88, E:87) — 100% (5/5), consistent high-volume output
3. PR Triage (Q:88, E:86) — 100% (1/1), structured reports
4. Auto-Triage Issues (Q:84, E:82) — 100% (5/5), fully stable
5. Avenger (Q:83, E:82) — 100% (2/2), proactive maintenance
6. Team Status (Q:82, E:81) — 1/1 success, excellent daily reports
7. Static Analysis (Q:81, E:80) — 1/1, 11+ days zero High findings
8. Agentic Maintenance (Q:80, E:78) — reliable, but 50% today (1/2)
9. Bot Detection (Q:76, E:76) — 1/1 success
10. AIC Consumption Report (Q:75, E:75) — 1/1, good observability

## P1 ESCALATION (NEW Jun 30)
- **PR Sous Chef**: 100% red (5/5 failures today, was 1/1 yesterday). Root cause: `gpt-5.5` routed via /chat/completions, gets 400. Issue #42444 OPEN. Fix tracked in #42421. Secondary harness bug: burns all 4 retries on non-retryable 400s.

## Persistent P1 Underperformers (DO NOT RE-FILE)
- Sub-Agent Model Resolution Audit: 100% red (Codex alpha 404). #42033 OPEN.
- PR Code Quality Reviewer: Tier-unsupported model. #42095 OPEN.
- Daily Safe Output Integrator: Tool denial 5/5. #42333 OPEN.
- Daily BYOK Ollama: api-proxy 503. #41827 OPEN.
- Go Logger Enhancement: jq ARG_MAX. #42032 OPEN.

## NEW FINDINGS (Jun 30)
- **Claude Code User Documentation Review**: cache-memory miss misconfiguration. #42482 filed (P2).
- **Daily Hippo Learn**: hippo MCP tool unavailable (0 tools exposed). #42442 filed (P2).
- **Harness retry waste**: burns all 4 retries on non-retryable 400 errors; documented in #42444.
- **Missing tool pattern**: Team Evolution (#42342) + Hippo Learn (#42442) — 2 agents, systemic.
- **CI integration test regression**: TestMCPGatewayDockerCommandUsesRunnerIdentityAndSocketGroup. #42423 OPEN (P1, filed by Workflow Health Mgr).

## Do Not Re-File (carry-forward)
#41827, #41987, #41988, #42032, #42033, #42095, #42329, #42332, #42333, #42342, #42356, #42370, #42398, #42421, #42423, #42444, #42471, #42482, #42442

## Engine Distribution (257 workflows)
- copilot: 158 (61%)
- claude: 60 (23%)
- pi: 20 (8%)
- codex: 15 (6%)
- other: 4 (1%)

## Coverage Gaps (carry-forward)
- No stale PR detection (PRs open >7d)
- No automated recovery/auto-close for persistent failures
- No AIC budget forecasting/alerting
