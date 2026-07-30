# ADR-49061: Detach Audit/Audit-Diff Subprocess Contexts from MCP Gateway Deadline

**Date**: 2026-07-30
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The MCP gateway imposes a hard per-tool RPC deadline (typically 60 seconds) on every inbound request context. The `audit` and `audit-diff` MCP tools passed this context directly to `exec.CommandContext`, which binds the subprocess lifetime to the gateway's deadline. As a result, every `audit` and `audit-diff` call failed with `context deadline exceeded` at exactly ~60 seconds, regardless of whether the actual workload could complete in time. The `logs` tool had already been fixed with the correct detach-and-own-timeout pattern; `audit` and `audit-diff` were inconsistently missing it.

### Decision

We will detach the `audit` and `audit-diff` subprocess contexts from the MCP gateway's RPC deadline by applying `context.WithoutCancel(ctx)` before creating a tool-owned `context.WithTimeout`. Context values (e.g. trace IDs) are preserved. A goroutine watcher selectively forwards only `context.Canceled` (explicit client disconnect) to the subprocess context — it never forwards `context.DeadlineExceeded` from the gateway, ensuring the subprocess can run for its full allotted time. This is identical to the pattern already used for the `logs` tool.

### Alternatives Considered

#### Alternative 1: Increase the MCP Gateway's RPC Deadline

Raise the gateway's per-tool timeout to 5+ minutes for all tools. This would prevent the 60 s kill but would require changes to gateway infrastructure/configuration outside this codebase. It also applies a blanket increase to all tools, including fast ones, where a 60 s deadline is a reasonable guard against runaway subprocesses.

#### Alternative 2: Redesign Tools for Incremental/Paginated Output

Break audit operations into smaller, faster incremental requests that each complete within the 60 s window. This would be a deeper protocol redesign with higher implementation complexity and would require changes on both the tool and client sides. It does not address the root cause of the subprocess-context coupling.

### Consequences

#### Positive
- `audit` and `audit-diff` tools no longer fail with `context deadline exceeded` after 60 seconds; legitimate long-running audits can complete within their 5-minute subprocess timeout.
- Client disconnects (explicit `context.Canceled`) still propagate promptly to clean up subprocesses, preventing orphaned processes.
- The pattern is now consistent across all three long-running privileged tools (`logs`, `audit`, `audit-diff`).

#### Negative
- A subprocess that has been detached from the gateway context can continue running for up to 5 minutes even if the gateway drops the connection for a reason other than explicit client cancellation (e.g. gateway restart, proxy timeout). Only `context.Canceled` is forwarded; `context.DeadlineExceeded` is intentionally suppressed.
- The 5-minute timeout constants (`defaultMCPAuditTimeoutMinutes`, `defaultMCPAuditDiffTimeoutMinutes`) are both set to the same value; if operational experience shows `audit-diff` needs more headroom than `audit` (it downloads artifacts for multiple runs), both constants must be updated independently.

#### Neutral
- Two goroutine watchers are introduced (one per tool), which is the same pattern already in use for `logs`. Each goroutine is bounded in lifetime by the subprocess context.
- Regression tests modeled on the existing `TestLogsToolSubprocessContextIgnoresGatewayDeadline` are added for both new tools.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
