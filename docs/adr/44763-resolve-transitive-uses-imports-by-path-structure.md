# ADR-44763: Resolve Transitive Uses Imports Using Path Structure Heuristic

**Date**: 2026-07-10
**Status**: Draft
**Deciders**: Unknown

---

### Context

The `gh aw add` command fetches workflow files and their `imports:` dependencies from a source repository. The internal function `fetchFrontmatterImportsRecursive` handles this recursively. Two bugs caused transitive imports to be silently dropped:

1. Bare filenames (e.g., `control-aux.md`, containing no `/`) were resolved against the workflow root (`originalBaseDir`) rather than the importing file's directory (`currentBaseDir`). This contradicted the compiler's own `determineNestedBaseDir` logic, producing incorrect remote paths and fetching from the wrong location.

2. When `force=false` and a shared file already existed on disk, the function skipped both the download and the recursion into that file's `imports:`. Any transitive dependencies referenced by the already-present file were never fetched.

A testability gap also existed: the production download function (`parser.DownloadFileFromGitHub`) was hard-coded, making it impossible to unit-test path resolution logic without real network calls.

### Decision

We will apply a path-structure heuristic in `fetchFrontmatterImportsRecursive` to choose the base directory for import resolution: paths **without** a `/` separator are resolved relative to `currentBaseDir` (the importing file's own directory), and paths **with** a `/` are resolved relative to `originalBaseDir` (the workflow root). This mirrors the compiler's `determineNestedBaseDir` logic.

We will also ensure that when a shared file already exists on disk and is skipped for re-download, the function reads the existing file and recurses into its imports before continuing — preserving the download-skip optimization while still completing transitive dependency resolution.

Additionally, we will introduce a `downloadFn` field on `frontmatterImportsOpts` that defaults to `parser.DownloadFileFromGitHub` when nil, enabling unit tests to inject a stub without network calls.

### Alternatives Considered

#### Alternative 1: Resolve all non-explicit paths against `currentBaseDir`

Resolve every non-explicit import path relative to the importing file's directory, regardless of whether the path contains `/`. This would be simpler and fully consistent.

This was not chosen because paths containing `/` (e.g., `shared/foo.md`) are authored by convention relative to the workflow root `.github/workflows`, not relative to the importing file's own location. Changing this convention would break all existing multi-segment cross-directory imports and would diverge from the compiler's resolution model, creating a split-brain between fetcher and compiler.

#### Alternative 2: Remove caching and always re-download (always use `force=true` behavior)

Remove the `force=false` optimization entirely so that every import is always re-downloaded, ensuring full recursion on every invocation without the need for the read-and-recurse fallback.

This was not chosen because it introduces unnecessary network round-trips for unchanged files. The skip-if-exists optimization is valuable for large workflow trees. The chosen fix is surgical: retain the skip, but add a read-and-recurse step on the existing file's content before continuing to the next import.

### Consequences

#### Positive
- Bare-filename imports (no `/`) now correctly resolve to the importing file's sibling directory, matching compiler behavior and eliminating silent resolution failures.
- Transitive dependencies are fetched even when intermediate shared files already exist on disk, ensuring complete dependency trees on every `gh aw add` invocation.
- The injected `downloadFn` field enables fast, hermetic unit tests for path resolution and recursion logic without network calls, consistent with the existing dispatch-workflow stub pattern in the codebase.

#### Negative
- Each pre-existing import file now incurs an additional `os.ReadFile` call during the skip path, increasing disk I/O proportionally to the number of already-installed shared files in large trees.
- The slash-based resolution heuristic is an implicit convention embedded in code rather than an explicit contract; future authors importing from unconventional directory structures may encounter unexpected behavior if their paths violate the `/`-presence assumption.

#### Neutral
- The `downloadFn` nil-check pattern (default to the real function when nil) is consistent with how other injectable functions are handled in the codebase.
- All existing paths containing `/` continue to resolve against `originalBaseDir`, so the change is backward-compatible with all current `imports:` declarations.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
