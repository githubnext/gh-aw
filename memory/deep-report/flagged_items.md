## Flagged Items for Monitoring (2026-01-26)

- GitHub remote MCP auth-test continues failing due to toolset loading; verify MCP server initialization and consider local fallback.
- CI lint failures from staticcheck QF1003 in pkg/campaign/interactive.go plus missing origin/main for incremental linting.
- list_code_scanning_alerts remains the largest MCP payload (24K tokens, 97KB); list_pull_requests also heavy due to duplicated repo objects.
- High-cost Copilot workflows (Agent Persona Explorer, CI Cleaner) remain outliers for tokens/run and total spend.
- Orchestration sessions still dominate Copilot session metrics, masking true task completion rates.
