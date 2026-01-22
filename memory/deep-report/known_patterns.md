## Known Patterns (2026-01-22)

- Token spend remains concentrated in CI Cleaner and Agent Persona Explorer; both exceed 6M tokens/run and together drive 25%+ of 30-day Copilot costs.
- MCP tooling reliability gaps persist: auth-test workflows continue to fail when MCP toolsets are missing in runtime.
- GitHub MCP payload bloat is recurrent in list_releases, list_pull_requests, and list_code_scanning_alerts; these endpoints exceed 3,800 tokens per call.
- Safe outputs maintain perfect health (100% success) across recent audits, indicating stable post-processing pipelines.
- Security posture remains strong; firewall escape tests continue to report zero new escapes post v0.9.1 patch.
- Workflow architecture is highly standardized: 100% concurrency controls, 74% scheduled, and dominant schedule+dispatch trigger pairing.
