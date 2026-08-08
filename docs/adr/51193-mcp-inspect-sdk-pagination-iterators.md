# ADR-51193: Use SDK Iterator-Based Pagination for MCP Inspect Capability Listing

**Date**: 2026-08-07
**Status**: Draft
**Deciders**: pelikhan (PR author), copilot-swe-agent (reviewer)

---

### Context

`gh aw mcp inspect` queries a connected MCP server for its capabilities (tools, resources, prompts) and displays them to the user. The previous implementation called `session.ListTools` and `session.ListResources` once with no cursor handling, silently truncating any results beyond the first page on servers that paginate their capability lists. Only `ListPrompts` had a manual cursor loop, creating an inconsistency. Additionally, the capability-querying logic was duplicated (~70 lines) between `connectStdioMCPServer` and `connectHTTPMCPServer`, differing only in transport setup. The MCP Go SDK v1.7.0 introduced iterator-based helpers (`session.Tools`, `session.Resources`, `session.Prompts`) that follow pagination cursors automatically.

### Decision

We will replace the single-shot `session.ListTools`/`session.ListResources` calls and the hand-rolled `ListPrompts` cursor loop with the SDK's iterator-based helpers (`session.Tools`, `session.Resources`, `session.Prompts`), and extract the shared listing logic into a single `queryServerCapabilities(ctx, config, session, verbose)` function called by both `connectStdioMCPServer` and `connectHTTPMCPServer`. Each listing operation runs in its own closure with a `defer`-scoped context cancel to ensure timeouts are contained.

### Alternatives Considered

#### Alternative 1: Add manual cursor loops to `ListTools` and `ListResources`

Extend the existing `ListPrompts` pattern to `ListTools` and `ListResources` by hand-rolling `for { result = session.ListXxx(ctx, &Params{Cursor: cursor}); if result.NextCursor == "" { break } }` loops. This would fix the truncation bug without adopting the new iterator API and would require no SDK version constraint change. It was rejected because it adds ~30 more lines of error-prone cursor-management boilerplate, keeps the logic duplicated across two connect functions, and diverges from the idiomatic SDK usage now available in v1.7.0.

#### Alternative 2: Keep single-shot calls, emit a warning on truncation

Detect a non-empty `NextCursor` in the single-shot response and log a warning ("results may be truncated") instead of fetching subsequent pages. This is simpler and avoids multi-page network round-trips for inspections. It was rejected because the `inspect` command's purpose is to show the complete server capability surface; a warning that silently drops data is a worse user experience than fetching all pages, especially since inspection is a one-shot diagnostic operation, not a hot path.

### Consequences

#### Positive
- MCP servers that paginate their tool/resource/prompt lists are now fully enumerated by `gh aw mcp inspect`, eliminating silent data loss.
- ~70 duplicate lines removed; the shared `queryServerCapabilities` helper is independently testable and covered by `TestQueryServerCapabilities_Pagination`.
- Per-listing context cancellation is now correctly scoped to each iterator closure, preventing timeout leaks if the listing logic changes in future.

#### Negative
- The iterator API (`for x, err := range session.Tools(...)`) requires Go 1.23+ range-over-function semantics; teams on older Go toolchains cannot compile this code.
- Error handling inside each iterator closure is break-on-first-error: a failure mid-page silently returns only the items collected so far, which may be harder to diagnose than a clear top-level error return.

#### Neutral
- The `connectStdioMCPServer` and `connectHTTPMCPServer` functions are now thinner, delegating all capability querying to the shared helper; their return type is unchanged (`*parser.MCPServerInfo, error`).
- This change has no effect on MCP servers that return all capabilities in a single page (the common case for small servers).

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
