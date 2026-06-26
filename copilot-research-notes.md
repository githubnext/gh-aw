# Copilot Research Notes (condensed)

## Persistent Gaps (never resolved across all runs)
- max-tool-denials: 0/63 SDK workflows (11+ consecutive runs)
- mcp.session-timeout: 0, mcp.tool-timeout: 0 (available since May 2026)
- engine.api-target: 0, engine.token-weights: 0
- blocked-domains: 0, engine.version genuine pinning: 0
- engine.args: 0 (11+ runs)
- startup-timeout: 1/252 (barely used)
- append-only-comments: 0 non-smoke uses (8 total, all in smoke tests)
- model-provider (non-GitHub): 0 non-smoke

## Key Trends (Jun-25 → Jun-26)
- engine.harness: 0→1 (new! daily-rendering-scripts-verifier.md)
- copilot_workflows: +3 (125→128)
- max-continuations: +1 (6→7)
- model_overrides: +3 (67→70)
- max-daily-ai-credits: +1 (73→74)
- web_fetch: -3 (29→26, slight regression)
- max-tool-denials: still 0 (no progress, 11+ consecutive runs)

## Historical Milestones
- cache-memory: 30 (May-21) → 72 (Jun-22) → 72 (Jun-25) → 72 (Jun-26) [plateau]
- max-daily-ai-credits: 73 (Jun-22) → 74 (Jun-26) [+1 slow growth]
- copilot-sdk: 63 since Jun-22, stable at 63 Jun-26
- engine.agent (custom): 8 (stable since Jun-22)
- model_overrides: 15 (May-21) → 42 (Jun-24) → 67 (Jun-25) → 70 (Jun-26) [strong growth]
- total workflows: 233 (May-21) → 251 (Jun-25) → 252 (Jun-26)
- copilot workflows: 100 (May-21) → 125 (Jun-25) → 128 (Jun-26)
- engine.harness: 0 (Jun-25) → 1 (Jun-26) [first use!]

## Engine Distribution (Jun-26)
- Copilot: 128/252 (51%) - 36 scalar + 86 object form
- Claude: 58/252 (23%)
- Pi: 19/252 (8%)
- Codex: 17/252 (7%)

## Agent Files Available (11 total in .github/agents/)
Used by workflows (6 unique agents, 8 total refs):
- adr-writer: 1 workflow (archie.md)
- ci-cleaner: 2 workflows (avenger.md, hourly-ci-cleaner.md)
- contribution-checker: 1 workflow
- technical-doc-writer: 2 workflows (glossary-maintainer.md, technical-doc-writer.md)
- agentic-workflows: referenced in workflow-generator.md
- developer.instructions: referenced in daily-file-diet.md

Unused (no workflow references):
- create-safe-output-type
- custom-engine-implementation
- grumpy-reviewer
- interactive-agent-designer
- w3c-specification-writer
