# Agent Performance — 2026-07-02T13:16Z | [§28592781498](https://github.com/github/gh-aw/actions/runs/28592781498)

## Scores: Q:61/100 (→) | E:62/100 (→) | Health:~72/100 (↓3, WHM outage)

## Top 10 Agents
| Agent | Q | E | Status |
|-------|---|---|--------|
| Copilot SWE Agent | 92 | 91 | 83% merge rate, 10+ PRs merged |
| Issue Monster | 88 | 87 | 100% |
| PR Triage Agent | 88 | 86 | On-time #42966 |
| Auto-Triage Issues | 84 | 82 | 100% |
| Avenger | 83 | 82 | 100% |
| Team Status | 82 | 81 | On-time #42948 |
| Static Analysis | 81 | 80 | On-time #42909 |
| Agentic Token Audit | 78 | 77 | On-time #42950 |
| AIC Consumption Report | 75 | 75 | On-time #42956 |
| Agentic Maintenance | 75 | 75 | 2/2 success |

## NEW Failures Today (15 [aw])
- #42908 Workflow Health Manager META-ORCHESTRATOR failed (repo-memory push)
- #42890 yamllint Fixer (3rd occurrence — escalate)
- #42918 GitHub Remote MCP Auth Test
- #42919 Multi-Device Docs Tester
- #42921 Sub-Agent Model Resolution Audit (carry-forward #42033)
- #42930 daily-experiment-report HTTP 400
- #42943 Daily Max Ai Credits Test (copilot engine crash)
- #42867 Matt Pocock Skills Reviewer (branch engine failure)
- #42872 Daily Credit Limit (6 AIC vs 1 threshold, 3rd recurrence)
- #42883 Daily Compiler Quality Check incomplete
- #42889 AI Moderator no safe outputs
- #42899 Smoke CI (#42398 carry-forward)
- #42960 Smoke Antigravity no safe outputs
- #42970 Impeccable Skills Reviewer
- #42971 Test Quality Sentinel

## P1 Persistent (DO NOT RE-FILE)
#42652 PR Sous Chef (pi-switch #42730 validating) · #42033 Model Audit · #42095 PR Code Quality · #42333 Safe Output Integrator · #41827 BYOK Ollama · #42032 Go Logger · #42423 CI Integration · #42824 Smoke Copilot Sub Agents · #42908 WHM (new)

## Key Findings
- WHM failed → health visibility gap today
- yamllint Fixer: 3 days consecutive → ESCALATE
- Credit limit recurring 3rd time (scheduling fix needed)
- PR Sous Chef pi-switch still validating
- Copilot SWE Agent: stable anchor
- Engine Distribution: copilot 61%, claude 23%, pi 8%, codex 6%

## Do Not Re-File
#41827,#41987,#41988,#42032,#42033,#42095,#42329,#42332,#42333,#42342,#42356,#42398,#42421,#42423,#42442,#42444,#42482,#42598,#42607,#42610,#42637,#42652,#42656,#42730,#42765,#42766,#42824,#42867,#42872,#42883,#42889,#42890,#42899,#42908,#42918,#42919,#42921,#42930,#42943,#42960,#42966,#42970,#42971
