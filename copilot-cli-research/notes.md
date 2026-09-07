# Copilot Research Notes (condensed) — updated 2026-09-07

## Persistent Gaps (unresolved across many runs)
- --share flag: 1 workflow (unchanged for months)
- --disable-builtin-mcps: 2 workflows only
- manual --add-dir usage: 1 workflow
- plugins: (Agent Plugins) config: 0 workflows use `plugins:` despite Copilot supporting it
- engine.version genuine pinning for copilot: rare
- engine.model override in copilot: still mostly alias-based (`small`, `openai/gpt-5.4`) rather than explicit copilot slugs

## Resolved/Improved since Aug-23
- custom agent files (`agent:`) usage jumped from 215 → 246 workflows — strong adoption
- repo-memory adoption grew from 2 → 38 workflows — big improvement
- gh-proxy mode (tools.github.mode) now used in 104 workflows — solid adoption
- sandbox config now in 230 workflows (up from earlier low counts)

## Historical Trend
- total_workflows: 233 (May-21) → 261 (Jul-25) → 289 (Aug-23) → 299 (Sep-07)
- copilot engine (broad incl. extended id/engine block): ~109/299 (36%)

## New Focus Areas (Sep-07 run)
- Agent Plugins (`plugins:`) feature has zero real-world adoption — worth a demo workflow
- --share flag still essentially unused; conversation-tracking benefit not being leveraged
- BYOK (COPILOT_PROVIDER_*) usage remains low (3 workflows) — mostly smoke tests
