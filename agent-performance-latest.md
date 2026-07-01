# Agent Performance — Latest Run

**Timestamp:** 2026-07-01T13:31Z | **Run:** [§28521103730](https://github.com/github/gh-aw/actions/runs/28521103730)

## Summary: 61/100 Quality (↓1) | 62/100 Effectiveness (↓1) | 75/100 Health (↓3)

## Top Performers
1. Copilot SWE Agent (Q:92, E:91) — 80% merge rate (68/85 PRs), highest-volume contributor
2. Issue Monster (Q:88, E:87) — 100%, consistent high-volume output
3. PR Triage (Q:88, E:86) — 100%, structured daily reports (#42760)
4. Auto-Triage Issues (Q:84, E:82) — 100% today (3/3), 9/10 this week
5. Avenger (Q:83, E:82) — 100%, proactive maintenance
6. Team Status (Q:82, E:81) — daily report #42744
7. Static Analysis (Q:81, E:80) — 11+ days zero High findings
8. AB Advisor (Q:78, E:76) — 2 issues today (#42732, #42733)
9. AIC Consumption Report (Q:75, E:75) — daily audit #42746 on-time
10. Content Moderation (Q:74, E:72) — 67% today (4/6)

## NEW THIS RUN
- **#aw_model_lifecycle (NEW P1 SYSTEMIC)**: Model version lifecycle management issue filed. gpt-5.5, codex alpha, claude-sonnet-5 all deprecated within one week; 3 P1s + 1 P2 are model-related. DO NOT RE-FILE.

## P1 Persistent Underperformers (DO NOT RE-FILE)
- PR Sous Chef: HTTP 400 recurring post-fix (#42652 OPEN). Engine switch to pi (#42730) active — validate next run.
- Sub-Agent Model Resolution Audit: codex alpha 404 (#42033 OPEN)
- PR Code Quality Reviewer: tier-unsupported model (#42095 OPEN)
- Daily Safe Output Integrator: tool denial 5/5 (#42333 OPEN)
- Daily BYOK Ollama: api-proxy 503 (#41827 OPEN)
- Go Logger Enhancement: jq ARG_MAX (#42032 OPEN)

## Do Not Re-File (carry-forward)
#41827, #41987, #41988, #42032, #42033, #42095, #42329, #42332, #42333, #42342, #42356, #42398, #42421, #42423, #42442, #42444, #42482, #42598, #42607, #42610, #42637, #42652, #42656, #42760, #42744, #42746

## Engine Distribution (257 workflows)
- copilot: 158 (61%)
- claude: 60 (23%)
- pi: 20 (8%)
- codex: 15 (6%)
- other: 4 (1%)

## Key Findings
- Model deprecation is now a systemic P1 concern (#aw_model_lifecycle)
- Harness burns 4 retries on non-retryable HTTP 400s — AIC waste
- PR Sous Chef engine-switch to pi in progress — validate
- Copilot SWE Agent: stable anchor at 80% merge rate
- 43 issues created today; 16 tagged [aw] (failures), remainder from healthy agents

## Coverage Gaps (carry-forward)
- No stale PR detection (PRs open >7d)
- No automated recovery/auto-close for persistent failures
- No AIC budget forecasting/alerting
- No proactive model version monitoring (→ #aw_model_lifecycle)
