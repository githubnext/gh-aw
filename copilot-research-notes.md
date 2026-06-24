# Copilot Research Notes (condensed)

## Persistent Gaps (never resolved across all runs)
- max-tool-denials: 0/63 SDK workflows (10+ consecutive runs)
- mcp.session-timeout: 0, mcp.tool-timeout: 0 (available since May 2026)
- engine.api-target: 0, engine.harness: 0, engine.token-weights: 0
- blocked-domains: 0, engine.version genuine pinning: 0
- engine.args: 0 (10+ runs; had 2 in May 21 run)
- startup-timeout: 1/251 (barely used)

## Key Trends (Jun-22 → Jun-24)
- min-integrity: REGRESSION 43→35 (-8 workflows)
- engine.driver: +3 (3→6 SDK custom drivers)
- pi engine: 19 workflows (growing)
- tracker-id: +1 (89→90)
- claude: +1 (56→57)
- codex: -1 (16→15)
- sandbox: -1 (23→22)

## Historical Milestones
- cache-memory: 30 (May-21) → 72 (Jun-22) → 72 (Jun-24) [plateau]
- max-daily-ai-credits: newly tracked at 73 in Jun-22, stable at 73 Jun-24
- copilot-sdk: not tracked in May, 63 since Jun-22
- engine.agent (custom): 8 (frontmatter strict count; previous runs used broader matching)

## Engine Distribution (Jun-24)
- Copilot: 121/251 (48%) - 35 scalar + 86 object form
- Claude: 57/251 (23%)
- Pi: 19/251 (8%)
- Codex: 15/251 (6%)

## Agent Files Available (11 total in .github/agents/)
Used by workflows (8 uses):
- adr-writer (archie.md)
- ci-cleaner (avenger.md, hourly-ci-cleaner.md)
- contribution-checker (contribution-check.md)
- developer.instructions (daily-file-diet.md)
- technical-doc-writer (glossary-maintainer.md, technical-doc-writer.md)
- agentic-workflows (workflow-generator.md)
Unused:
- create-safe-output-type
- custom-engine-implementation
- grumpy-reviewer
- interactive-agent-designer
- w3c-specification-writer
