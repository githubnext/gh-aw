## Known Patterns (2026-01-23)

- Token spend remains concentrated in CI Cleaner and Agent Persona Explorer; top 2 workflows drive 34%+ of Copilot cost and exceed 7M tokens/run.
- MCP payload bloat persists for list_code_scanning_alerts and list_pull_requests; token-heavy tools dominate MCP response budgets.
- MCP tool availability gaps recur in auth-test runs (tool bindings missing in runtime), causing consistent failures before server contact.
- Safe outputs are generally healthy, but add_comment 404s can appear when discussions are deleted between agent output and handler execution.
- Copilot session completion rate is improving (28% today), but orchestration/PR review sessions still dominate skips and action_required outcomes.
- Workflow corpus remains highly standardized: 139 lock files, 100% concurrency usage, and workflow_dispatch+schedule as the dominant trigger pairing.
