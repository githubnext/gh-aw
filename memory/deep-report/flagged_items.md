## Flagged Items for Monitoring (2026-01-22)

- MCP remote authentication test failures due to missing MCP toolsets in runtime (auth-test discussion 2026-01-22).
- High token usage per run in CI Cleaner and Agent Persona Explorer; optimization needed to reduce cost exposure.
- GitHub MCP tool response bloat (list_releases, list_pull_requests, list_code_scanning_alerts) creates context pressure for agents.
- Elevated skip rate (50%) in Copilot session insights; indicates heavy orchestration filtering and reduced executable task sample.
- Schema documentation drift: removed included_file_schema still referenced in docs; safe-jobs missing from schema.
