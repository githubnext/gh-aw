## Known Patterns (2026-01-16)

- Firewall escape surfaced via `docker exec` into sibling safe-outputs container; indicates firewall enforcement only on agent container and no proxy/iptables in sibling containers (critical security gap).
- Safe-outputs validation errors recur in two workflows: Changeset Generator emitting empty `update_pull_request` and Issue Monster using `target=triggering` in scheduled runs, causing repeated add_comment failures.
- GitHub API access gaps persist in Copilot PR merged reporting (missing safeinputs-gh/unauthenticated GH token), blocking daily summary generation.
- MCP GitHub tool definitions show schema/param issues (invalid schema for `github-get_commit`, missing owner param for `github-list_code_scanning_alerts`).
- Token consumption remains concentrated in Issue Monster and CI Cleaner; high per-run costs still dominate the 30-day baseline.
- Issue flow shows closures outpacing creation over last 3 days, but unlabeled issues remain (27 total, 5 open).
