# Copilot Research Notes (condensed)

## Persistent Gaps (never resolved across all runs)
- max-tool-denials: 0/63 SDK workflows (10+ consecutive runs)
- mcp.session-timeout: 0, mcp.tool-timeout: 0 (available since May 2026)
- engine.api-target: 0, engine.harness: 0, engine.token-weights: 0
- blocked-domains: 0, engine.version genuine pinning: 0
- engine.args: 0 (10+ runs)
- startup-timeout: 1/251 (barely used)

## Key Trends (Jun-24 → Jun-25)
- model_overrides: +25 (42→67, SIGNIFICANT growth)
- copilot_scalar: +2 (35→37 scalar form)
- copilot_workflows: +4 (121→125)
- web_fetch: +7 (22→29)
- min-integrity: still 35 (regression from 43 in Jun-22 UNRESOLVED)
- max-tool-denials: still 0 (no progress, 10+ consecutive runs)
- engine.driver: ~5-6 (stable)
- pi engine: 19 workflows (stable)

## Historical Milestones
- cache-memory: 30 (May-21) → 72 (Jun-22) → 72 (Jun-24) → 72 (Jun-25) [plateau]
- max-daily-ai-credits: 73 since Jun-22, stable at 73 Jun-25
- copilot-sdk: 63 since Jun-22, stable at 63 Jun-25
- engine.agent (custom): 8 (stable since Jun-22)
- model_overrides: 42 (Jun-24) → 67 (Jun-25) [major jump]
- total workflows: 249→251 (Jun-20→Jun-24), 251 (Jun-25) [plateau]
- copilot workflows: 122→121 (Jun-20→Jun-24), 125 (Jun-25) [slight recovery]

## Engine Distribution (Jun-25)
- Copilot: 125/251 (50%) - 37 scalar + 86 object form
- Claude: 57/251 (23%)
- Pi: 19/251 (8%)
- Codex: 15/251 (6%)

## Agent Files Available (11 total in .github/agents/)
Used by workflows (6 unique agents, 8 total refs):
- adr-writer: 1 workflow (archie.md)
- ci-cleaner: 2 workflows (avenger.md, hourly-ci-cleaner.md)
- contribution-checker: 1 workflow
- technical-doc-writer: 2 workflows (glossary-maintainer.md, technical-doc-writer.md)
- agentic-workflows: referenced in workflow-generator.md
- developer.instructions: referenced somewhere

Unused (no workflow references):
- create-safe-output-type
- custom-engine-implementation
- grumpy-reviewer
- interactive-agent-designer
- w3c-specification-writer
