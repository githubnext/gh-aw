# ADR-43834: Split remote_fetch.go into Focused Single-Responsibility Modules

**Date**: 2026-07-06
**Status**: Draft
**Deciders**: Unknown

---

### Context

`pkg/parser/remote_fetch.go` had grown to 1,553 lines across five unrelated functional domains: shared HTTP client utilities, local include-path resolution and security validation, WorkflowSpec parsing and download dispatch, ref-to-SHA resolution with multi-tier auth fallbacks, single-file download with symlink handling, and directory listing (workflow files, flat, recursive, subdirs). A single file spanning five concerns made it hard to navigate, review, and isolate in tests. Merge conflicts were more frequent because unrelated features competed for changes in the same file.

### Decision

We will split `remote_fetch.go` into six focused files, all residing in the same `parser` package, each under 450 lines and responsible for exactly one functional domain: `remote_client.go` (shared HTTP/REST utilities), `remote_resolve_path.go` (include-path resolution and security validation), `remote_workflow_spec.go` (WorkflowSpec parsing, SHA caching, download dispatch), `remote_resolve_sha.go` (ref→SHA resolution with auth/git/public-API fallback chain), `remote_download_file.go` (single-file download with symlink resolution and three-tier fallback), and `remote_list_files.go` (directory listing). All exported symbols are preserved unchanged; no logic is altered.

### Alternatives Considered

#### Alternative 1: Keep the monolithic file with internal comment sections

Add region-style comments (e.g., `// --- HTTP Client ---`) to demarcate the five domains within `remote_fetch.go` without splitting files. This is zero-risk from a compilation and test perspective and avoids any directory-structure change. It was not chosen because comment markers are non-enforced conventions — they drift and get ignored under deadline pressure, and navigating a 1,500-line file still requires IDE search. The size problem recurs the moment a new domain is added.

#### Alternative 2: Extract into separate sub-packages under `pkg/parser/remote/`

Create sub-packages such as `pkg/parser/remote/client`, `pkg/parser/remote/download`, etc., moving each domain into its own package. This enforces separation at the language level (the compiler will catch circular imports) and makes inter-domain dependencies explicit. It was not chosen because it would require changing all call sites across the `parser` package, potentially introduce circular imports given the current tight coupling, and is a larger refactor scope than the immediate problem warrants. Same-package file splits achieve the navigability goal with zero API breakage.

### Consequences

#### Positive
- Each file is under 450 lines with a single clearly named responsibility, making targeted code review and debugging faster.
- Merge conflicts on `remote_fetch.go` are eliminated; changes to download logic no longer race with changes to SHA resolution or directory listing.
- Easier to unit-test each domain in isolation without loading the other four domains into view.
- New contributors can discover where a specific function lives by filename rather than grepping a 1,500-line file.

#### Negative
- The `pkg/parser` directory now contains more files (six new files replace one), which may feel noisy at a glance before the reader understands the naming convention.
- The same-package split does not enforce domain boundaries at compile time — a function in `remote_download_file.go` can still call unexported helpers in `remote_list_files.go` without the compiler flagging it.

#### Neutral
- No logic changes mean all existing tests pass without modification; behavior for external callers of exported symbols (`DownloadFileFromGitHub`, `ListWorkflowFiles`, etc.) is identical.
- The wasm build stub `remote_fetch_wasm.go` is untouched; the build-tag-gated split is consistent with that pattern.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
