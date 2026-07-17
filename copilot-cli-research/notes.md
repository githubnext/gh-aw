# Copilot CLI Research Notes

## Run History

| Date | Run ID | Total WFs | Copilot WFs | Key Findings |
|------|--------|-----------|-------------|--------------|
| 2026-07-16 | 29472014909 | 257 | 125 | engine.driver at 0, mcp-scripts at 1, model overrides at 10 |
| 2026-07-17 | 29555564313 | 258 | 125 | engine.driver at 6, mcp-scripts at 3, model overrides at 79, block-domains at 0 |

## Persistent Zero-Usage Features (Never Adopted)

These features have been at 0 usage for multiple consecutive runs:
- `engine.args` - Custom CLI arguments (--add-dir, --verbose, etc.)
- `engine.cwd` - Working directory override
- `engine.harness.max-attempts` / `engine.harness.delay` - Retry tuning
- `network.blocked` / `block-domains` - Domain blocklist (new feature, 0 adoption)

## Positive Trends

- `engine.driver`: 0→6 (new pattern emerging, 3 languages: Python, Node, TypeScript)
- Model overrides: 10→79 (massive adoption in 1 day - likely bulk addition)
- Version pinning: 18→38 (doubled)
- `bare` mode: 11→21 (+10)
- `mcp-scripts`: 1→3 (+2, recovering from regression)

## Recommendations Tracking

| Recommendation | Status | Issue |
|----------------|--------|-------|
| Add tracker-id to daily workflows | Pending | Created 2026-07-17 issue |
| Add block-domains to security workflows | Pending | Created 2026-07-17 issue |
| max-continuations for long workflows | Pending | Created 2026-07-17 issue |
| Harness retry tuning | Pending | Created 2026-07-17 issue |
| LSP servers for code analysis workflows | Pending | Created 2026-07-17 issue |
| Document engine.driver pattern | Pending | Created 2026-07-17 issue |
