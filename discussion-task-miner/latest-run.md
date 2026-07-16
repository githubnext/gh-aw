# Task Mining Run - 2026-07-16

## Summary
- Discussions scanned: 13 (new since last run)
- Tasks identified: 5
- Issues created: 0 (create_issue limit already exhausted by previous runs this cycle)
- Duplicates avoided: 0

## Tasks Identified (ready for next run)
1. **Add compile-time assertions for CodingAgentEngine types** — 8 concrete types in `pkg/workflow/*_engine.go` — Source: #45742
2. **Add compile-time assertions for ConditionNode implementors** — 10 expression node types — Source: #45742
3. **Introduce MCPTransportType named Go type** — replace ~49 scattered `"stdio"/"http"/"local"` literals — Source: #45727
4. **Consolidate duplicate safe-output config structs** — 6 files repeat same embedded struct shape — Source: #45727
5. **Replace `{Raw any}` tool-config wrappers with shared RawToolConfig** — 3 identical structs in `tools_types.go` and `repo_memory.go` — Source: #45727

## Top Patterns Observed
- Compile-time interface assertions: 0 production assertions for 16+ interfaces (2 discussions)
- Duplicate struct shapes: 6 identical safe-output config types, 3 identical raw-tool-config types
- Untyped enums: MCPTransportType (~49 sites), LLMProvider (~31 sites)

## Note
Issue creation was blocked (limit reached). Tasks above should be created in next run.
