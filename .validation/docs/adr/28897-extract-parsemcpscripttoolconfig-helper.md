# ADR-28897: Extract `parseMCPScriptToolConfig` Helper to Eliminate Parsing Duplication

**Date**: 2026-04-28
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The MCP scripts parser (`pkg/workflow/mcp_scripts_parser.go`) contained two separate ~100-line blocks that each parsed a single `MCPScriptToolConfig` from a `map[string]any`: one inside `parseMCPScriptsMap` and one inside `mergeMCPScripts`. The blocks were structurally identical, covering description, inputs/params, script/run/py/go, env, and timeout parsing. When a `uint64` integer overflow bug was discovered in the timeout handling, it had to be patched at both sites independently (alerts #413 and #414), demonstrating that the duplication was an active maintenance hazard. Any future addition of an `MCPScriptToolConfig` field would require two separate edits with no compiler-enforced guarantee of consistency.

### Decision

We will extract a single private helper function `parseMCPScriptToolConfig(toolName string, toolMap map[string]any) *MCPScriptToolConfig` that contains the entire parsing logic for one MCP script tool config entry. Both `parseMCPScriptsMap` and `mergeMCPScripts` will call this helper instead of duplicating the logic inline. This makes `MCPScriptToolConfig` parsing a single, auditable code path that is modified in one place when the struct evolves.

### Alternatives Considered

#### Alternative 1: Accept the Duplication

Keep both inline parsing blocks and rely on code-review discipline to keep them in sync. This was the status quo prior to this PR. It was rejected because the `uint64` overflow fix showed that even careful reviewers missed the need to update both sites — the duplication caused a real defect that required two separate security-alert fixes.

#### Alternative 2: Parsing via a Configuration-Driven Strategy or Interface

Introduce an interface (e.g., `MCPToolParser`) or a table-driven approach where parsing rules are declared as data rather than code. This would be more extensible if the tool config format were highly variable or plugin-driven. It was not chosen because the config format is well-defined and stable; the added complexity of an interface brings no benefit over a simple concrete helper function for this use case.

### Consequences

#### Positive
- A single canonical code path for parsing `MCPScriptToolConfig` eliminates the risk of the two sites drifting apart again.
- Future additions to `MCPScriptToolConfig` (new fields, validation logic) require only one edit, and the compiler enforces that both callers receive the updated behaviour automatically.

#### Negative
- `parseMCPScriptToolConfig` is a package-private function; it is not exported and therefore cannot be easily tested in isolation by external test packages. Tests must exercise it through the higher-level callers.
- The extraction adds one level of indirection when reading either call site — a reader must follow the function call to understand full parsing behaviour.

#### Neutral
- Net code change is −97 lines (205 deletions, 108 additions), reducing the size of the parser file significantly.
- The timeout `uint64` overflow safe-conversion comment is now consolidated into a single location, referencing both original alerts (#413 and #414).

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### MCP Script Tool Config Parsing

1. All parsing of a single `MCPScriptToolConfig` from a raw `map[string]any` **MUST** be performed by the `parseMCPScriptToolConfig` helper function.
2. Implementations **MUST NOT** duplicate the per-tool parsing logic inline within `parseMCPScriptsMap`, `mergeMCPScripts`, or any future caller.
3. The `parseMCPScriptToolConfig` function **MUST** apply all field parsing (description, inputs, script, run, py, go, env, timeout) and **MUST** return a fully initialised `*MCPScriptToolConfig` with all map fields initialised to non-nil defaults.
4. Integer overflow-safe conversion of `uint64` timeout values **MUST** use `typeutil.SafeUint64ToInt` within `parseMCPScriptToolConfig` as the single authoritative conversion site.

### Callers

1. `parseMCPScriptsMap` **MUST** delegate per-tool config construction to `parseMCPScriptToolConfig` rather than constructing `MCPScriptToolConfig` values inline.
2. `mergeMCPScripts` **MUST** delegate per-tool config construction to `parseMCPScriptToolConfig` rather than constructing `MCPScriptToolConfig` values inline.
3. Any new function that needs to produce an `MCPScriptToolConfig` from a raw map **SHOULD** call `parseMCPScriptToolConfig` rather than re-implementing the parsing logic.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25047928528) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
