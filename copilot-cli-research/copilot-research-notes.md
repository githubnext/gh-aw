# Copilot Research Notes (condensed)

## Persistent Gaps (never resolved across all runs, confirmed again Aug-23)
- max-tool-denials: 0 copilot workflows explicit (18+ consecutive runs — most persistent critical gap)
- engine.token-weights: still 0 (no progress since first tracked)
- engine.api-target: 0
- network.blocked: effectively 0 (only in smoke-claude.md)
- engine.version genuine pinning for copilot: 0
- tools.startup-timeout: 1 workflow only
- engine.model override in copilot: 0 (still using model: aliases predominantly)
- --share / --add-dir(manual) / --disable-builtin-mcps: near-zero adoption (1, 1, 2 workflows respectively) — NEW finding, tracked from Aug-23 run

## Resolved/Improved
- mcp-scripts: grew from 1 to 3 workflows
- bare-mode: grew to 21 (mostly other engines)
- web-fetch: 25 → still ~3 among Copilot-specific workflows sampled Aug-23 (broader repo count higher)
- cache-memory: 30 (May-21) → 94 (Jul-25) → 10 among Copilot-specific workflows sampled Aug-23
- engine.agent (custom): grew significantly to 215 workflows

## Key Trends (Jul-23 → Aug-23)
- total_workflows: 261 → 289 (+28)
- copilot_explicit: 32 → 37 → 39 (+2 since Jul-25)
- repo-memory adoption: still low (2 workflows) — new area of focus for next run

## Historical Milestones
- cache-memory: 30 (May-21) → 94 (Jul-25)
- copilot-sdk: 66 (stable since Jun)
- engine.agent (custom): grew significantly to 215 workflows
- total workflows: 233 (May-21) → 261 (Jul-25) → 289 (Aug-23)

## Engine Distribution (Aug-23)
- Copilot (explicit): 39/289 (13.5%)
- Claude: 37 (12.8%)
- Codex: 9 (3.1%)
- Default (empty engine:): majority remainder

## Agent Files Available (in .github/agents/)
- create-safe-output-type: UNUSED (needs re-check)
- custom-engine-implementation: UNUSED (needs re-check)
- w3c-specification-writer: check usage

## New Focus Areas (Aug-23 run)
- repo-memory vs cache-memory adoption gap for trend-tracking workflows
- lack of shared Copilot engine-defaults snippet
- model slug inconsistency (alias vs concrete slug)
