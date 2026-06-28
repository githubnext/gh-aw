# Copilot Research Notes (condensed)

## Persistent Gaps (never resolved across all runs)
- max-tool-denials: 0/64 SDK workflows (12+ consecutive runs)
- engine.api-target: 0, engine.token-weights: 0
- network.blocked: 0 (engine-level domain blocking)
- engine.version genuine pinning: 0
- engine.args: 0 (only in smoke tests)
- tools.startup-timeout: ~1/257 (barely used)
- tools.timeout: ~1/257
- append-only-comments: 0 non-smoke uses
- model-provider (non-GitHub): 0 non-smoke

## Key Trends (Jun-26 → Jun-28)
- copilot_workflows: 128→126 (counting method diff)
- sdk_workflows: 63→64 (+1)
- max_tool_denials: still 0 (no progress, 12+ consecutive runs)
- total_workflows: 252→257 (+5)

## Historical Milestones
- cache-memory: 30 (May-21) → 72 (Jun-22) → 72 (Jun-25) → 44 (Jun-28, counting change)
- copilot-sdk: 63 (Jun-22, Jun-26) → 64 (Jun-28) [slow growth]
- engine.agent (custom): 8 (stable since Jun-22)
- total workflows: 233 (May-21) → 251 (Jun-25) → 257 (Jun-28)
- copilot workflows: 100 (May-21) → 128 (Jun-26) → 126 (Jun-28)
- engine.harness: 0 (Jun-25) → 1 (Jun-26) → 1+ (Jun-28)

## Engine Distribution (Jun-28)
- Copilot: 126/257 (~49%) - scalar + object form
- Claude: ~58 (~23%)
- Pi: ~19 (~7%)
- Codex: ~17 (~7%)

## Agent Files Available (11 total in .github/agents/)
Used by workflows (6 unique agents):
- adr-writer: archie.md
- ci-cleaner: avenger.md, hourly-ci-cleaner.md
- contribution-checker: 1 workflow
- technical-doc-writer: glossary-maintainer.md, technical-doc-writer.md
- agentic-workflows: workflow-generator.md
- developer.instructions: daily-file-diet.md

Unused (no workflow references):
- create-safe-output-type
- custom-engine-implementation
- grumpy-reviewer
- interactive-agent-designer
- w3c-specification-writer

## Top Optimization Opportunities
1. max-tool-denials on ALL 64 SDK workflows (HIGH, prevents infinite tool denial loops)
2. tools.timeout for MCP stability (HIGH, prevents hanging workflows)
3. startup-timeout on all workflows with MCP servers (MEDIUM)
4. network.blocked for security-sensitive workflows (MEDIUM)
5. max-continuations on long-timeout workflows (MEDIUM, autopilot)
6. Adopt unused agent files (MEDIUM, grumpy-reviewer, w3c-specification-writer)
7. Version pinning for stable/release workflows (MEDIUM)
8. engine.api-target for GHEC/GHES readiness (LOW)
9. BYOK mode for cost optimization (LOW)
