# Task Mining Run - 2026-08-05 (19:08 UTC)

## Summary
- Discussions scanned: 22 (unprocessed since last run, IDs 50520-50671)
- Tasks identified: 6
- Issues created: 5
- Duplicates avoided: 1 (error-message actionability findings already filed as #50591/#50592)

## Created Issues
- Consolidate 4 duplicated MCP-server-stats structs in pkg/cli into a shared base
- Consolidate 3 duplicated per-tool aggregate call stats structs in pkg/cli
- Strengthen getParsedSchemaDoc return type from any to map[string]any in pkg/parser
- Type BeforeState/AfterState audit fields as MutableItemState instead of map[string]any
- Migrate two tabwriter tables in pkg/cli/logs_format_compact.go to console.RenderTable

## Top Patterns Observed
- pkg/cli audit/gateway reporting subsystem has repeated semantic-duplicate structs (MCP stats, tool stats) — same root cause across 4 clusters, most already have an existing dedup pattern (`AnalysisBase`) to follow
- `any`/`map[string]any` used where a concrete shape is fully known (schema docs, audit item state) — easy, high-value typing wins
- Several reports (session-insights, daily-status, PR-merged-report, auto-triage, secrets scan, cache-strategy, GEO audit) were pure status/metrics reports with no extractable code-quality tasks
- Error message actionability audit (#50563) reproduced findings already tracked as issues #50591/#50592 — skipped to avoid duplication
