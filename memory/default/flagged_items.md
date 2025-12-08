## Flagged Items for Monitoring (2025-12-08)

- Repeated workflow failures: Daily Copilot PR Merged Report (two recent failures) and Weekly Issue Summary (failure) — check data dependencies and auth to reduce noise.
- Super Linter Report failed in latest window — verify rule set or recent code changes causing lint breaks.
- High error count (183 across 20 runs) despite modest costs — investigate noisy logging or flaky tool calls.
- Label dominance (`ai-generated`, `plan`) may mask urgent human-authored issues; consider filtering or auto-triage for untagged/urgent items.
- Daily issue spikes (Dec 8) — ensure alerting to avoid backlog growth; open issues still 56 (22% of weekly volume).
