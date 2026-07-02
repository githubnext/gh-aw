# ADR-42875: Split pkg/parser/remote_fetch.go into Focused Responsibility Modules

**Date**: 2026-07-02
**Status**: Draft
**Deciders**: Unknown (automated refactor by copilot-swe-agent)

---

### Context

`pkg/parser/remote_fetch.go` had grown to approximately 1547 lines of Go code combining four distinct concerns: include-path and workflow-spec resolution, ref/SHA resolution (with GitHub API and git ls-remote fallbacks), file download (with API, raw URL, git archive, and git clone fallbacks), and directory/workflow listing (with API and git-based fallbacks plus a process-lifetime clone cache). The mixed responsibilities made it difficult to locate, understand, and maintain any individual fallback chain. Future contributors needed to scroll past hundreds of unrelated lines to find the function they were debugging.

### Decision

We will split `remote_fetch.go` into four focused files within the same `parser` package, each owning one cohesive concern:

- `remote_fetch.go` — include-path resolution, workflow-spec resolution, shared module-level vars (`remoteLog`, `publicAPIClient`)
- `remote_fetch_refs.go` — ref/SHA resolution and its API, git ls-remote, and public-API fallbacks
- `remote_fetch_download.go` — file download, symlink resolution, and the API, raw URL, git archive, git clone, and public-API fallback chain
- `remote_fetch_listing.go` — workflow-file and directory listing, git-clone–based listing cache, and all listing fallbacks

All exported entry points (`DownloadFileFromGitHub*`, `ResolveRefToSHAForHost`, `ListWorkflowFiles*`, `ListDir*`) are preserved intact and the fallback ordering is unchanged.

### Alternatives Considered

#### Alternative 1: Keep the Single File, Add Section Markers

Retain `remote_fetch.go` as-is but organize it with prominent comment banners dividing the four concern groups. This approach avoids any structural change and carries zero risk of accidental behavior change from moving code.

Not chosen because section markers are a convention that erodes without enforcement — future additions are inevitably dropped into the wrong section — and the file would continue growing. Navigation still requires manual scrolling past the entire file.

#### Alternative 2: Extract Remote-Fetch Logic into a Separate Package

Move the four concern groups out of `pkg/parser` into a new package (e.g., `pkg/remotefetch`) with explicit exported interfaces that `pkg/parser` calls.

Not chosen because it would require adding exported symbols for functions that are currently unexported but shared across the four files (e.g., `createRESTClientForHost`, `buildContentsAPIPath`), increasing the public API surface area. The added package boundary overhead was judged to exceed the maintenance benefit for an intra-parser subsystem.

### Consequences

#### Positive
- Each file's scope is immediately clear from its name; navigating to a specific fallback requires opening one file instead of scrolling through 1547 lines.
- Independent concern files are easier to review in isolation — a change to ref resolution does not touch download or listing code.
- Build-tag isolation (`//go:build !js && !wasm`) can now be applied per concern file rather than to the entire module.

#### Negative
- Files remain in the same package, so unexported symbols are still freely visible across files; the split does not enforce interface boundaries and does not prevent future cross-concern coupling from creeping back in.
- Tracing a complete end-to-end call path (e.g., from `DownloadFileFromGitHub` through fallbacks) now requires navigating across two files (`remote_fetch_download.go` calls helpers in `remote_fetch.go`), which can be less obvious than a single-file read.

#### Neutral
- No behavioral changes — existing tests, exported API signatures, and fallback sequencing are identical before and after.
- The `docs/adr/` file for this PR was generated automatically from the diff; the Deciders field and any context the PR author considered but did not commit to the diff should be completed before changing Status from Draft to Accepted.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
