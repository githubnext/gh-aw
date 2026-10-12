# ADR-68051: Pin explicit MCP protocol versions and cache only discovery responses

**Date**: 2026-10-12
**Status**: Draft
**Deciders**: Unknown (draft generated from PR #68051)

---

### Context

The `gh-aw` MCP server (`pkg/cli/mcp_server.go`) is constructed with `mcp.NewServer` from the Model Context Protocol Go SDK and, until now, relied entirely on SDK defaults for two client-visible behaviours. First, the set of protocol versions the server would negotiate was whatever `mcp.SupportedProtocolVersions()` happened to return for the pinned SDK version, so an SDK bump could silently drop support for an older client (for example an agent still speaking `2024-11-05`) with no compile-time or test-time signal. Second, the server advertised no cache hints at all: clients re-issued `server/discover` and `tools/list` on every session even though the tool set is static (`Tools.ListChanged` is already `false`). Separately, the `compile` and `work-queue` tools shell out to the CLI and can run for many seconds, but unlike other long-running tools in the codebase they emitted no progress notifications, leaving MCP clients with no feedback during the call. The constraint is that none of these changes may alter negotiated behaviour for existing clients or change tool result payloads.

### Decision

We will make all three behaviours explicit at the server construction site. We will list the five supported protocol versions (`2026-07-28` through `2024-11-05`) literally in `mcp.ServerOptions.SupportedProtocolVersions`, so that dropping or adding a version becomes a deliberate, reviewable code change rather than a side effect of an SDK upgrade; a table-driven test iterates `mcp.SupportedProtocolVersions()` and asserts each version still negotiates successfully. We will add a `SetCacheable` hook that attaches a `private`, 60-second TTL hint (`mcpDiscoveryCacheTTLMs`) to **only** `server/discover` and `tools/list` responses — tool call results are deliberately left uncached because they reflect mutable repository state. Finally, we will call the existing `notifyProgress` helper at the start and end of the `compile` and `work-queue` handlers, which is a no-op unless the client supplied a progress token.

### Alternatives Considered

#### Alternative 1: Keep relying on SDK defaults for protocol versions

Not passing `SupportedProtocolVersions` means the server automatically tracks whatever the SDK supports, which is less code and guarantees the server never advertises a version the SDK cannot actually serve. It was rejected because gh-aw is consumed by agent runtimes whose MCP client versions we do not control, and an SDK bump silently narrowing the negotiable range would surface only as client-side initialization failures in the field. Pinning the list is a close call — it adds a maintenance obligation on every SDK bump — but the accompanying test over `mcp.SupportedProtocolVersions()` turns that obligation into a failing test rather than a production regression.

#### Alternative 2: Cache tool results as well as discovery responses

A broader `SetCacheable` policy covering `tools/call` would give larger latency wins, since `compile` and `work-queue` are the expensive operations. It was rejected because those results are functions of the working tree and the work-queue git state, which change between calls within a single session; serving a 60-second-stale compile result would be a correctness bug, not a performance win. Restricting caching to the static tool list and discovery metadata captures the safe subset of the benefit.

#### Alternative 3: Stream fine-grained progress from the subprocess

Instead of two fixed `0` and `100` notifications, the handlers could parse CLI stdout/stderr incrementally and report per-workflow progress. It was rejected for this change because it would require the CLI commands to emit a structured progress stream and would couple the MCP layer to CLI output formats; bookend notifications deliver the main benefit (the client knows work started and finished) at negligible cost.

### Consequences

#### Positive
- Protocol-version support is explicit and test-covered, so an SDK upgrade that changes the negotiable range fails CI instead of breaking clients silently.
- Clients can skip redundant `server/discover` and `tools/list` round trips for 60 seconds, reducing startup chatter for a tool set that is already declared static.
- MCP clients now get start/completion feedback for the two longest-running tools, matching the progress behaviour of other tools.

#### Negative
- The hard-coded version list must be updated by hand on every SDK bump; forgetting to add a newly supported version means gh-aw quietly lags the SDK.
- A 60-second private cache window means a client may use a stale tool list; this is currently safe only because `ListChanged` is `false`, so any future dynamic tool registration would have to revisit this hint.
- `createMCPServer` grew enough to require a `//nolint:largefunc` suppression, deferring a real decomposition of that constructor.

#### Neutral
- `CacheScope: "private"` prevents shared/proxy caching, so the hint only helps the connecting client, not fleet-wide callers.
- `notifyProgress` is a no-op without a client-supplied progress token, so clients that do not request progress see no behavioural change.
- Progress is reported as a fixed `0`/`100` pair rather than a real completion percentage.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
