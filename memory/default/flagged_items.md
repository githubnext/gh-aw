## Flagged Items for Monitoring (2025-12-23)

- Burst of failures from branch `copilot/create-custom-action-setup-activation` (multiple workflows failed in one push: Plan Command, Release, Q, Nitpick Reviewer, DeepReport, Scout, Spec-Kit dispatcher, etc.); needs root-cause to stop failure noise.
- Daily Copilot PR Merged still reports a missing tool (run §20464285031) despite success; confirm tool wiring/safeinputs.
- Firewall denial rate now ~30% with GitHub API and LinkedIn top blocks; Copilot workflows still lack GitHub MCP config and LinkedIn access intent is unclear.
- High-cost hourly CI Cleaner continues to dominate Copilot token spend (~29% of cost over 4 days); consider frequency reduction or no-op exit.
- Unlabeled issues grew to 20 total (2 open); maintain triage to prevent drift.

## Flagged Items for Monitoring (2025-12-22)

- Weekly Issue Summary workflow failing (last 7d logs show single failure among 10 runs); needs triage to restore weekly reporting.
- Missing GitHub MCP read_issue capability in Dev workflow causing missing-tool error (run §20435819459); add tool or adjust permissions.
- Daily Copilot PR Merged needs safeinputs-gh tool to avoid missing-tool failures (run §20435787142); ensure workflow wiring matches instructions.
- Issue backlog uptick: open issues climbed to 45 (was 21 on 12/20) with 85 created in last 3 days; watch for sustained growth.
- Unlabeled issues remain (19 total, 5 open); consider triage to prevent drift.

## Flagged Items for Monitoring (2025-12-20)

- Safe outputs env var miss: Daily Issues Report generator lacks `GH_AW_ASSETS_BRANCH`, causing the lone safe_outputs failure; needs workflow config fix.
- AI Moderator failures: Multiple consecutive AI Moderator issue_comment runs failed (e.g., [§20401747970](https://github.com/githubnext/gh-aw/actions/runs/20401747970), [§20401612678](https://github.com/githubnext/gh-aw/actions/runs/20401612678), [§20401610443](https://github.com/githubnext/gh-aw/actions/runs/20401610443), [§20401406395](https://github.com/githubnext/gh-aw/actions/runs/20401406395)); investigate trigger inputs/permissions.
- Copilot GitHub access: Firewall report shows ongoing `api.github.com` blocks; Copilot workflows still need consistent GitHub MCP server configuration instead of direct API calls.
- Schema mismatch: Runtime accepts string `max-turns` while schema is integer-only; update schema or tighten runtime for consistency.
- Onboarding gaps: Doc noob test highlights missing “what is this tool” framing, glossary/tooling definitions, and clearer lockfile guidance; actionable doc quick wins remain open.
