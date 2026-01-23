## Flagged Items for Monitoring (2026-01-23)

- sandbox.mcp.port lacks runtime validation despite schema constraints; critical gap open 15+ days.
- tools.timeout and tools.startup-timeout minimums are not enforced at runtime; risk of invalid values.
- MCP auth-test failures due to missing MCP tool bindings in runtime (not server auth errors).
- Safe output add_comment 404s from deleted discussions should be downgraded to warnings to avoid false failure rates.
- list_code_scanning_alerts remains an outlier for MCP payload size and usefulness score.
- Issue Monster shows high failure rate despite frequent runs; investigate error clusters if trend persists.
