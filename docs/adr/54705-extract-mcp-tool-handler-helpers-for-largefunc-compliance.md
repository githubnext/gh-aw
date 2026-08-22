# ADR-54705: Extract MCP Tool Handler Helpers for `largefunc` Compliance

**Date**: 2026-08-22
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The `pkg/cli/mcp_tools_privileged.go` file registers three MCP tools (logs, audit, audit-diff). Each registration function passed a large anonymous closure directly to `mcp.AddTool`. The logs handler alone was 193 lines; the overall `registerLogsTool` function reached 244 lines. The repository enforces a `largefunc` custom linter (max-lines=60) that blocks `make golint-custom` for any file containing functions exceeding this limit. These two files (`pkg/cli/mcp_tools_privileged.go` and `pkg/console/progress.go`) were the last remaining violations in a broader backlog. No public API, MCP tool schema, or behavior changes were involved.

### Decision

We will decompose the overlong registration functions and handler closures in `pkg/cli/mcp_tools_privileged.go` and `pkg/console/progress.go` into named package-level helper functions, moving each logical concern (argument building, error envelope construction, empty-result construction, handler factory) into its own named function. The logs handler's inline gateway-deadline-detaching goroutine will be replaced by the existing `newMCPSubprocessContext` helper already used by the audit tools, eliminating ~35 lines of duplicated context-management logic.

### Alternatives Considered

#### Alternative 1: Linter Suppression (`//nolint:largefunc`)

Add `//nolint:largefunc` annotations to the overlong functions instead of refactoring. This approach is zero-risk for behavioral regressions and requires only a one-line change per function. It was rejected because it defeats the purpose of the linter — the `largefunc` rule was added specifically to prevent overly large functions that are harder to review and test, and suppressing it perpetuates the underlying readability problem without addressing it.

#### Alternative 2: Struct-Based Encapsulation

Introduce a handler struct (e.g., `logsToolHandler`) whose fields hold `execCmd`, `actor`, and `validateActor`, and whose methods implement `buildArgs`, `buildEmptyResult`, etc. This provides better namespacing and aligns with an object-oriented style, but adds new named types and method-receiver indirection for what is effectively a small collection of utilities that are unlikely to be reused outside this file. The added complexity is not justified for a pure compliance refactor.

### Consequences

#### Positive
- All functions in the two touched files now comply with the `largefunc` (max-lines=60) rule, unblocking `make golint-custom` for this slice of the backlog.
- The logs tool reuses `newMCPSubprocessContext`, eliminating ~35 lines of duplicated deadline-detaching goroutine logic and making the two tool handlers consistent.
- Helper functions (`buildLogsCommandArgs`, `buildAuditCommandArgs`, `resolveAuditRunItems`, etc.) are independently testable units.

#### Negative
- More top-level names are introduced in `pkg/cli`, making it harder to follow the end-to-end flow of a single tool registration by reading one contiguous function.
- The inline comment block that explained the MCP gateway deadline-detaching rationale is removed; readers must navigate to `newMCPSubprocessContext` to understand the context-management invariant.

#### Neutral
- No changes to MCP tool schemas, public API surfaces, or observable behavior; the refactor is internal to the registration and handler plumbing.
- The repo-wide `make golint-custom` continues to report findings for functions outside these two files; this PR addresses only one slice of the backlog.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
