# ADR-49968: Add Prompt Discovery to `mcp-inspect`

**Date**: 2026-08-03
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The `mcp-inspect` command queries MCP servers and reports their advertised capabilities — currently tools and resources. MCP servers can also advertise *prompts*, a first-class capability type in the MCP protocol, but `mcp-inspect` never called `ListPrompts`, so servers that exposed prompts appeared incomplete or were silently under-reported. The `MCPServerInfo` data model in `pkg/parser/mcp.go` similarly had no field for prompts, meaning the information could not flow through the existing inspect pipeline even if queried.

### Decision

We will extend the MCP inspection flow to include prompts as a first-class capability type. `MCPServerInfo` gains a `Prompts []*mcp.Prompt` field; both `connectStdioMCPServer` and `connectHTTPMCPServer` call `session.ListPrompts` with the existing `MCPOperationTimeout`; and `displayServerCapabilities` renders an "Available Prompts" table (name, title, description, argument count) in the same style as tools and resources. Prompt discovery is best-effort — a `ListPrompts` error emits a verbose warning and continues rather than failing the whole inspection.

### Alternatives Considered

#### Alternative 1: Keep prompts out of scope; document the limitation

The `mcp-inspect` command could explicitly document that it only shows tools and resources, treating prompts as out-of-scope for the current inspection surface. This avoids expanding the struct and any extra network calls, but leaves users with an incomplete picture of MCP server capabilities and no path to discover prompt names or arguments without a custom client.

#### Alternative 2: Gate prompt discovery behind an opt-in flag (`--show-prompts`)

Prompts could be fetched only when the user explicitly passes `--show-prompts`, limiting the extra network round-trip to cases where it is wanted. This reduces default latency but increases command-line complexity and means prompts are invisible by default — inconsistent with how tools and resources are always shown.

### Consequences

#### Positive
- Users see a complete representation of MCP server capabilities in a single `mcp-inspect` invocation; no extra flags needed.
- The best-effort error-handling model (verbose warning on failure, continue) is consistent with the existing behavior for tools and resources.

#### Negative
- Every `mcp-inspect` run now issues an additional `ListPrompts` RPC, adding latency even when the inspected server has no prompts.
- `MCPServerInfo` grows a new field; any code that constructs this struct (tests, callers) must initialise `Prompts` or risk a nil slice where an empty slice is expected.

#### Neutral
- The `MCPOperationTimeout` constant comment is updated to mention `ListPrompts` alongside `ListTools` and `ListResources`, reflecting the expanded scope.
- Tests added for both the prompts-present and prompts-absent rendering paths, keeping coverage aligned with the new code paths.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
