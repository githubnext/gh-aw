# Copilot Research Notes (condensed)

## Persistent Gaps (never resolved across all runs)
- max-tool-denials: 0/65 SDK workflows (14+ consecutive runs — most persistent critical gap)
- engine.api-target: 0, engine.token-weights: 0
- network.blocked: effectively 0 (only in smoke-claude.md, not a real Copilot workflow)
- engine.version genuine pinning: 0
- tools.timeout: 0 workflows
- tools.startup-timeout: 1 workflow (ruflo-backed-task only)
- engine.copilot-sdk-driver: 0

## Key Trends (Jun-28 → Jul-05)
- sdk_workflows: 64 → 65 (+1)
- total_workflows: 257 → 258 (+1)
- copilot effective (default+explicit): ~126 → 202 (measurement refined)
- max_tool_denials: still 0 (no progress, 14+ runs)
- engine.harness: 1 (stable)
- grumpy-reviewer: now used in designer-drift-audit.md
- interactive-agent-designer: used in designer-drift-audit.md
- append-only-comments: 8 workflows (first tracked)
- mcp-scripts: 18 workflows (first tracked)

## Historical Milestones
- cache-memory: 30 (May-21) → 72 (Jun-22) → 44 (Jun-28) → 96 (Jul-05, counting all engines)
- copilot-sdk: 63 (Jun-22) → 64 (Jun-28) → 65 (Jul-05)
- engine.agent (custom): 8 → 20 (significant growth!)
- total workflows: 233 (May-21) → 257 (Jun-28) → 258 (Jul-05)
- copilot effective: 100 (May-21) → ~126 (Jun-28) → 202 (Jul-05, refined counting)

## Engine Distribution (Jul-05, top-level workflows)
- Copilot (explicit): 38/258 (15%)
- Copilot (default 'engine:' empty): 137 + implicit
- Claude: 47 (18%)
- Codex: 10 (4%)
- No engine field (default copilot): many

## Agent Files Available (11 in .github/agents/)
Used by workflows:
- adr-writer: 4 workflows
- agentic-workflows: 241 workflows (AGENTS.md effectively)
- ci-cleaner: 2 workflows
- contribution-checker: 2 workflows
- developer: 36 workflows
- grumpy-reviewer: 2 workflows (designer-drift-audit.md) 
- interactive-agent-designer: 2 workflows (designer-drift-audit.md)
- technical-doc-writer: 4 workflows

Unused (no workflow references):
- create-safe-output-type: 0
- custom-engine-implementation: 0
- w3c-specification-writer: 0

## Top Optimization Opportunities (Jul-05)
1. max-tool-denials on ALL 65 SDK workflows (HIGH, 14+ runs unchanged, prevents infinite loops)
2. tools.timeout for MCP stability (HIGH, only 0 production workflows)
3. copilot-requests:write adoption: 31/38 explicit Copilot, 81/137 default Copilot missing
4. startup-timeout on all MCP-using workflows (MEDIUM, only 1 workflow uses it)
5. max-continuations on long-timeout workflows: 13 Copilot workflows have 30+ min timeout without it
6. engine.token-weights for AI credit budgeting (MEDIUM, 0 production despite feature availability)
7. Adopt w3c-specification-writer and create-safe-output-type agent files
8. network.blocked for security-sensitive workflows (MEDIUM)
9. engine.version genuine pinning for stable/release workflows (LOW)
