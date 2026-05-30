# ADR-35812: Decompose Claude Allowed-Tools Assembly into Focused Helpers

**Date**: 2026-05-29
**Status**: Draft
**Deciders**: Unknown

---

## Part 1 — Narrative (Human-Friendly)

### Context

`ClaudeEngine.computeAllowedClaudeToolsString` in `pkg/workflow/claude_tools.go` had grown into a single ~360-line function that assembled the `--allowed-tools` string for the Claude engine. It interleaved many unrelated responsibilities in one scope: neutral→Claude tool conversion and defaulting, Claude `allowed` extraction (including Bash wildcard handling via `goto` labels), top-level MCP/cache-memory permission expansion, sandbox writable-path normalization, safe-outputs and mcp-scripts grants, and final de-duplication. The deep nesting, `goto nextClaudeTool` control flow, and repeated `slices.Contains` guards made the function a flagged high-complexity hotspot in the ongoing "lint-monster" effort to reduce large-function complexity across the workflow/CLI codepaths, and the embedded wildcard/path-normalization logic was not independently unit-testable.

### Decision

We will decompose `computeAllowedClaudeToolsString` into a pipeline of focused, single-responsibility helper functions — `prepareClaudeToolsForAllowedList`, `collectClaudeAllowedTools`, `appendTopLevelClaudeTools`, `appendSandboxWritableTools`, `appendSafeOutputsTools`, `appendMCPScriptsTools`, and `dedupeAllowedTools` — each transforming the running `allowedTools` slice, with the orchestrator reduced to a linear sequence of calls followed by the existing sort/join. This is a strictly **behavior-preserving** refactor: the emitted `--allowed-tools` string remains sorted and de-duplicated exactly as before. The primary driver is reducing cyclomatic complexity for maintainability; a secondary benefit is extracting pure logic (`hasBashWildcard`, `normalizeSandboxWritablePattern`, `getOrCreateToolMap`, `appendIfMissing`) into units covered by new helper-level unit tests.

### Alternatives Considered

#### Alternative 1: Leave the function as-is or suppress the complexity lint

Keep the monolithic function and silence the complexity warning with a suppression directive. Rejected because the function is on a hot, security-sensitive path (it decides which tools an agent is permitted to invoke), and its `goto`-driven control flow and untestable inner logic make it error-prone to modify. Suppressing the lint preserves the maintenance hazard the lint exists to surface.

#### Alternative 2: Introduce a builder/struct that accumulates permissions

Model the assembly as a stateful `allowedToolsBuilder` type with mutating methods (`.addDefaults()`, `.addMCP()`, etc.) instead of free functions threading a `[]string`. Rejected for this PR because it is a larger change that introduces new state and lifecycle, increasing the surface area beyond a behavior-preserving extraction; the free-function pipeline keeps each step pure and independently testable while matching the existing procedural style of the file. A builder remains a reasonable future evolution if more permission sources are added.

### Consequences

#### Positive
- Each permission source (Claude allowed, cache-memory, MCP, sandbox, safe-outputs, mcp-scripts) lives in a named helper, so a contributor changing one concern reads and edits only that helper rather than navigating a 360-line function.
- Pure helpers (`hasBashWildcard`, `normalizeSandboxWritablePattern`) are now unit-tested in `claude_tools_helpers_test.go`, locking the wildcard and path-normalization semantics against regressions during subsequent complexity-reduction work.
- The `goto nextClaudeTool` control flow is eliminated in favor of early-returning helpers, and repeated `!slices.Contains(...)` guards are consolidated behind `appendIfMissing`.

#### Negative
- The number of package-level functions in `claude_tools.go` grows substantially, and the running `allowedTools` slice is now threaded through many `append*` calls, which a reader must follow across functions to reconstruct the full output.
- Extraction introduces new exported-within-package surface (`getOrCreateToolMap`, `appendIfMissing`, etc.) that future contributors may reuse inconsistently if the intended scope is not understood.

#### Neutral
- The output contract is unchanged by design, so no caller, snapshot, or generated lock file should differ; correctness rests on the existing higher-level tests plus the new helper tests.
- `appendIfMissing` centralizes the dedup-on-append idiom, but a final `dedupeAllowedTools` pass is still retained to catch duplicates arising from wildcard/non-wildcard bash normalization, so the two mechanisms coexist intentionally.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Output Contract Preservation

1. The refactored `computeAllowedClaudeToolsString` **MUST** produce a byte-for-byte identical `--allowed-tools` string to the pre-refactor implementation for any given `(tools, safeOutputs, cacheMemoryConfig, mcpScripts, sandboxConfig)` input.
2. The final output **MUST** remain sorted alphabetically and de-duplicated.
3. The refactor **MUST NOT** add, remove, or rename any tool permission that the prior implementation would have emitted.

### Decomposition Structure

1. The orchestrator function **MUST** delegate each distinct permission-source concern (Claude `allowed` extraction, top-level/MCP permissions, cache-memory, sandbox writable paths, safe-outputs, mcp-scripts, de-duplication) to a dedicated helper function.
2. Helper functions that accumulate permissions **SHOULD** accept and return the running `allowedTools` slice rather than mutating shared package state.
3. Pure decision logic extracted from the orchestrator (bash wildcard detection, sandbox path normalization) **MUST** be implemented as standalone functions with no side effects.
4. The implementation **SHOULD NOT** reintroduce `goto`-based control flow for tool iteration.

### Test Coverage

1. Extracted pure helpers (`hasBashWildcard`, `normalizeSandboxWritablePattern`) **MUST** have direct unit tests covering wildcard, non-wildcard, and rejection (relative/empty path) cases.
2. The pre-existing higher-level tests for `computeAllowedClaudeToolsString` **MUST** continue to pass unchanged.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26667328855) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
