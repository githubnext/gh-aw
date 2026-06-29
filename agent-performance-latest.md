# Agent Performance — Latest Run

**Timestamp:** 2026-06-29T14:01Z | **Run:** [§28377470831](https://github.com/github/gh-aw/actions/runs/28377470831)

## Summary: 63/100 Quality (→ stable) | 64/100 Effectiveness (→ stable) | 80/100 Health (↓2)

## Top Performers
1. Copilot SWE Agent (Q:92, E:91) — 80% merge rate (61/76 settled), highest-volume contributor
2. PR Triage (Q:88, E:86) — consistent structured reports, 1/1 success today
3. Team Status (Q:85, E:83) — 1/1 success, excellent daily health reports
4. Static Analysis (Q:84, E:81) — 1/1 success, 11+ days zero High findings
5. Workflow Health Manager (Q:82, E:80) — accurate P1/P2 tracking, good coordination
6. Agentic Maintenance (Q:80, E:82) — 3/3 success, 100% reliable
7. Auto-Triage Issues (Q:78, E:80) — 2/2 success, fully recovered (was P1 Jun 21–25)
8. Bot Detection (Q:75, E:78) — 1/1 success
9. PR Sous Chef (Q:74, E:76) — 1/1 success, consistent

## Persistent P1 Underperformers (DO NOT RE-FILE)
- Sub-Agent Model Resolution Audit: 100% red (Codex alpha 404). Issue #42033 OPEN.
- PR Code Quality Reviewer: Tier-unsupported model. Issue #42095 OPEN.
- Daily Safe Output Integrator: Tool denial 5/5. Issue #42125 OPEN.
- Daily BYOK Ollama: api-proxy 503. Issue #41827 OPEN.
- Go Logger Enhancement: jq ARG_MAX. Issue #42032 OPEN.

## NEW FINDINGS (Jun 29)
- **Escalating tool denial pattern**: Layout Spec Maintainer (#42204) joins Safe Output Integrator (#42125) and Formal Spec Verifier (#42105) — 3 agents now hitting tool denial limit. Systemic issue filed: #aw_toolden1
- **AI Moderator no-safe-output**: #42234 filed (0/4 success, 2 action_req, 2 skipped)
- **GitHub MCP Structural Analysis**: #42248 filed for missing tool — latest run SUCCEEDED (1/1), may be intermittent
- **Q workflow high action_required rate**: 10/14 (71%) action_required — likely by design (awaiting human dispatch), monitor

## Do Not Re-File (carry-forward)
Code Simplifier issues CLOSED Jun 28. Do not re-file: #41827, #41987, #41988, #42032, #42033, #42095, #42105, #42124, #42125, #42128, #42140, #42204, #42234.

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
