# PR Triage Run History

| Run ID | Date | Agent PRs | Fork PRs Eligible | Triaged | Closed Since | Notes |
|--------|------|-----------|-------------------|---------|--------------|-------|
| §28258543430 | 2026-06-26T18:51Z | 14 | 0 | 0 | 10 | 13 new PRs, #41762 very large (+1622/-833), #41295 oldest (47h) |
| §28239513901 | 2026-06-26T12:59Z | 11 | 0 | 0 | 4 | 9 new PRs (burst), draft-heavy run |
| §28224135185 | 2026-06-26T07:37Z | 6 | 0 | 0 | 2 | 4 new PRs (burst activity), #41623 high-risk open |
| §28210925990 | 2026-06-26T01:20Z | 3 | 0 | 0 | 5 | #41555 (draft bump), #41553 (refactor), #41295 (draft fix) |
| §28193371345 | 2026-06-25T18:58Z | — | 0 | 0 | — | prior run |

## Observations
- All Copilot PRs use same-repo branches (`copilot/*`); no fork PRs have been seen across 5 consecutive runs
- Fork-only triage policy consistently results in 0 eligible PRs
- Agent is extremely active: creating 10+ PRs per run cycle, closing them quickly
- High throughput: 10 PRs closed in ~6h between last two runs
- #41295 is the only persistently open PR (47h+) — a bug fix for apply_samples
- Typical PR lifecycle: draft → ready → closed within 1-6 hours
