# Task Mining Run - 2026-08-01

## Summary
- Discussions scanned: 30
- Tasks identified: 8
- Issues created: 2
- Duplicates/already-tracked avoided: 6

## Created Issues
- refactor: migrate remaining RunGHWithHost call sites to RunGHContextWithHost (89% still legacy) — from discussion #49349
- refactor: extract shared DifcContentFields struct in pkg/cli/gateway_logs_types.go — from discussion #49330 (Typist)

## Top Patterns Observed
- gh CLI wrapper context-propagation gap mostly closed by prior PR #48488, but RunGHWithHost (cross-host ops) still 89% legacy — new focused issue created
- Typist duplicate-type clusters: most high-value clusters (updateFailure, CopilotWorkflowStep, RepositoryFeatures wasm pair, TokenCoreMetrics embedding, mcp_github_config/mcp.go typed wiring) already have open or closed issues; only the DIFC content-fields cluster was untracked
- Schema/docs drift (antigravity engine), lint-monster function-length backlog, and most daily audit reports are already tracked by dedicated recurring issues
