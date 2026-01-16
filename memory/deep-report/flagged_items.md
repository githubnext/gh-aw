## Flagged Items for Monitoring (2026-01-16)

- CRITICAL: Firewall escape via `docker exec` into safe-outputs container allows unrestricted outbound access (discussion #10180); requires immediate network isolation/proxy enforcement.
- MCP GitHub server schema error for `github-get_commit` causes AI Moderator failure (discussion #10199).
- Missing GitHub API access in copilot-pr-merged report (discussion #10319) blocks daily PR analysis.
- Recurring safe-output validation failures in Issue Monster (scheduled add_comment target) and Changeset Generator (empty update_pull_request) (discussion #10145).
- Changeset Generator run failed with missing agent logs; needs visibility into codex agent failures (discussion #10199).
