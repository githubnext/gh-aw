# ADR-43128: Thread context.Context Through Public Remote Fetch APIs

**Date**: 2026-07-03
**Status**: Draft
**Deciders**: Unknown

---

### Context

The 8 public APIs in `pkg/parser/remote_fetch.go` (`DownloadFileFromGitHub`, `DownloadFileFromGitHubForHost`, `ResolveRefToSHAForHost`, `ListWorkflowFiles`, `ListWorkflowFilesForHost`, `ListDirAllFilesForHost`, `ListDirAllFilesRecursivelyForHost`, `ListDirSubdirsForHost`) and all their private helpers hard-coded `context.Background()` throughout their call chains. This made it impossible for CLI commands to propagate cancellation signals: pressing Ctrl-C could not interrupt in-flight HTTP requests because no cancellable context was reachable by the HTTP layer. Go's `context.Context` idiom requires passing a caller-supplied context as the first parameter of I/O-bound functions so that the caller can cancel, time out, or add deadlines. Without this, each CLI command that fetches remote resources silently blocks until the HTTP stack times out on its own, degrading the interactive user experience.

### Decision

We will add `ctx context.Context` as the first parameter to all 8 public remote fetch APIs and propagate it through every private helper in the same call chain, replacing all 7 hard-coded `context.Background()` call sites. All CLI callers are updated to pass `cmd.Context()` (the Cobra command context, which is cancelled on Ctrl-C). Two remaining hard-coded `context.Background()` calls inside `downloadIncludeFromWorkflowSpec` / `resolveWorkflowSpecSHAForCache` — which sit behind `ResolveIncludePath` across 6+ files in 4 packages — are deferred to a follow-up PR because threading context through that compile-time include-resolution path requires a broader coordinated change.

### Alternatives Considered

#### Alternative 1: Keep context.Background() — Status Quo

The simplest option is to leave all call sites as-is. This avoids any API surface change. It was rejected because it permanently prevents signal propagation: Ctrl-C cannot cancel HTTP calls, long-running fetches block indefinitely on poor connections, and there is no path to adding per-operation timeouts in the future without the same refactor.

#### Alternative 2: Use a Package-Level or Global Cancellable Context

A global or package-level `context.Context` could be set at startup and read by `remote_fetch.go` without changing any function signatures. This avoids the API churn. It was rejected because it couples the library to a mutable global state, makes behaviour non-deterministic in tests (shared state across goroutines), and is explicitly against the Go standard library's context design guidelines, which require context to flow through the call graph as an explicit argument.

#### Alternative 3: Wrap the HTTP Client with a Default Timeout Instead

Adding a fixed timeout (e.g., 30 s) to the `http.Client` used internally would automatically abort hung requests. This would be simpler than threading context everywhere. It was rejected because a fixed timeout cannot be adjusted by callers, cannot be cancelled early on user interrupt (Ctrl-C fires before the timeout), and does not compose with higher-level deadlines or tracing. It addresses the symptom rather than the root cause.

### Consequences

#### Positive
- CLI commands (e.g., `gh aw add`, `gh aw list`) now respond to Ctrl-C by cancelling in-flight HTTP requests immediately, eliminating hung-process behaviour.
- The codebase now follows standard Go context idioms throughout the remote I/O layer, making future work (per-command timeouts, distributed tracing, cancellation propagation) straightforward.
- `cmd.Context()` from Cobra is wired end-to-end, meaning any future context metadata (deadlines, trace IDs) attached by callers is automatically available to all remote fetch operations.

#### Negative
- The change touches 15 files and ~20+ function signatures across `pkg/parser/`, `pkg/cli/`, and related packages; it is a wide refactor with high merge-conflict potential for concurrent branches.
- Two `context.Background()` call sites inside the compile-time include-resolution path (`downloadIncludeFromWorkflowSpec`, `resolveWorkflowSpecSHAForCache` via `ResolveIncludePath`) are intentionally deferred, leaving partial context propagation in that sub-system until a follow-up PR.

#### Neutral
- Callers that previously called the public APIs without a context must now supply one; any future consumer of this library (internal or external) must comply with the updated signatures.
- The private wrapper functions in `add_package_manifest.go` (e.g., `downloadPackageFileFromGitHubForHost`, `listPackageDirFilesForHost`) gained `ctx` parameters alongside the public API changes; these are not breaking changes to exported symbols but do affect internal package coupling.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
