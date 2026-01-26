## Known Patterns (2026-01-26)

- Copilot token spend remains concentrated: Agent Persona Explorer (30.2M) and CI Cleaner (26.9M) are top cost drivers, while Code Scanning Fixer runs most frequently with high efficiency.
- MCP response bloat persists for list_code_scanning_alerts (24K tokens, 97KB) and list_pull_requests (13.8K tokens), while labels/branches/workflows/discussions remain consistently efficient.
- GitHub remote MCP auth-test failures continue to be tool-loading issues (toolsets not available), not authentication failures.
- CI reliability issues are dominated by lint failures (staticcheck QF1003) and missing origin/main for incremental linting; core workflow success rates remain high otherwise.
- Copilot session completion rate improved to 20% with strong error-recovery success (90.9%), while orchestration sessions still skew overall completion metrics.
- Workflow corpus continues steady growth with strict standards: 139 lockfiles, 100% concurrency, 70% schedule + workflow_dispatch pairing, and consistent 50-100 KB sizes.
