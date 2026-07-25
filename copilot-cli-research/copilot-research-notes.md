# Copilot Research Notes (condensed)

## Persistent Gaps (never resolved across all runs)
- max-tool-denials: 0 copilot workflows out of 37 explicit (17+ consecutive runs — most persistent critical gap)
- engine.token-weights: still 0 (no progress since first tracked)
- engine.api-target: 0
- network.blocked: effectively 0 (only in smoke-claude.md)
- engine.version genuine pinning for copilot: 0
- tools.startup-timeout: 1 workflow only
- engine.model override in copilot: 0

## Resolved/Improved
- mcp-scripts: grew from 1 to 3 workflows
- bare-mode: grew to 21 (mostly other engines)
- web-fetch: 25 workflows (grew significantly last run)

## Key Trends (Jul-23 → Jul-25)
- total_workflows: 261 unchanged
- copilot_explicit: 32 → 37 (+5)
- mcp_scripts: 1 → 3 (+2)
- max_tool_denials: still 0 (no progress)

## Historical Milestones
- cache-memory: 30 (May-21) → 94 (Jul-25)
- copilot-sdk: 66 (stable since Jun)
- engine.agent (custom): grew significantly to 215 workflows
- total workflows: 233 (May-21) → 261 (Jul-25)

## Engine Distribution (Jul-25)
- Copilot (explicit): 37/261 (14%)
- Claude: 43 (16%)
- Codex: 9 (3%)
- Default (empty engine:): ~145

## Agent Files Available (11 in .github/agents/)
- create-safe-output-type: UNUSED
- custom-engine-implementation: UNUSED
- w3c-specification-writer: NEW - check usage
