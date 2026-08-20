# ADR-53895: Consolidate Duplicated Wasm/Native Parser Helpers into Build-Tag-Free Files

**Date**: 2026-08-19
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

`pkg/parser` maintained separate implementations of five path-predicate helpers — `isUnderWorkflowsDirectory`, `isCustomAgentFile`, `isRepositoryImport`, `IsWorkflowSpec`, and the path-arithmetic helpers `findGitHubFolder` / `computeIncludeResolveAndSecurityBases` — duplicated across `remote_fetch_wasm.go` (build tag `js || wasm`) and `remote_resolve_path.go` / `remote_workflow_spec.go` (build tag `!js && !wasm`). These copies were not thin platform shims; they contained identical business logic, and they had already drifted: the wasm copy of `isRepositoryImport` rejected any repo name containing a dot (`strings.Contains(repo, ".")`), while the native copy rejected only known data-file extensions (`.md`, `.yaml`, `.yml`, `.json`). As a result, a valid import like `githubnext/gh-aw.dev` was accepted on native builds and silently rejected on wasm. The wasm copy also hardcoded string literals (`".github/workflows/"`, `".github/agents/"`) that the native copy correctly referenced via package constants. The repository already contained the right pattern for this situation: `github_token_env.go` is a build-tag-free file that both `github.go` (native) and `github_wasm.go` (wasm) delegate to for shared env-var logic.

### Decision

We will extract all platform-independent path predicates and path-arithmetic helpers into a new build-tag-free file `pkg/parser/remote_path_predicates.go` (plus `path_section.go` for the `path#section` splitter), standardize on the native `isRepositoryImport` semantics (extension-based rejection, not dot-based), and delete the duplicated implementations from the build-tag-specific files. Only the filesystem-probe step (which genuinely differs: `os.Stat` vs. `VirtualFileExists`) remains in the platform-specific files.

### Alternatives Considered

#### Alternative 1: Keep both copies in sync with comments and documentation

Add comments in both files and a CONTRIBUTING note reminding developers to update both the wasm and native copy whenever either changes. This requires zero structural change and no risk of behavioral regression.

Why not chosen: the drift in `isRepositoryImport` demonstrates this approach already failed. Cognitive load on every future contributor, with no enforcement mechanism, means further drift is certain. The behavioral bug would recur.

#### Alternative 2: Define a platform abstraction interface

Define a `PathPredicates` interface and have each build target provide an implementation, allowing each platform to override specific predicates with full type safety. This is the more formal Go pattern for build-tag polymorphism.

Why not chosen: none of the five helpers in question have any platform-specific behavior after the `isRepositoryImport` semantics choice is made. An interface with five methods, two concrete implementations that are byte-identical, and zero intentional divergence adds pure indirection with no benefit. It also increases the surface area for future drift by providing a slot where platform differences *could* be introduced even when they shouldn't be.

#### Alternative 3: Remove the wasm build target

Eliminate the wasm build entirely to remove the need for any build-tag split, resolving the drift problem permanently.

Why not chosen: wasm is a key deployment target for browser-based gh-aw use cases. Removing it is not on the roadmap.

### Consequences

#### Positive
- Eliminates approximately 120 lines of duplicated code across three files and two build targets.
- Behavioral divergence between native and wasm for `isRepositoryImport` is fixed; dotted repository names like `githubnext/gh-aw.dev` are now accepted consistently.
- Constants (`WorkflowsDirSlash`, `AgentsDir`, `GithubDir`) are used in both build targets, so future constant changes propagate to both automatically.
- Follows an established codebase pattern (`github_token_env.go`) rather than introducing a new one.

#### Negative
- The build-tag-free file is exercised by the native test suite only; wasm-specific behavior that relied on the old (more restrictive) `isRepositoryImport` predicate could surface as a behavior change in the wasm build without a dedicated wasm test run.
- `computeIncludeResolveAndSecurityBases` is now shared and called from both `remote_resolve_path.go` and `remote_fetch_wasm.go`; any future divergence between the two platforms' security-boundary logic must be handled at the call site rather than by forking the function.

#### Neutral
- The three byte-identical `path#section` splitters (`splitImportPathAndSection`, `splitIncludePathAndSection`, `stripImportSection`) are replaced by a single `splitPathAndSection` helper, consistently using `strings.Cut` instead of `strings.SplitN`.
- `extractFrontmatterForTopologicalSort` in `import_topological.go` is replaced by a direct call to `extractFrontmatterForImport`, reducing the number of frontmatter extraction helpers from two to one.
- `isWorkflowSpec` (the wasm-only alias calling `IsWorkflowSpec`) is deleted; call sites now reference `IsWorkflowSpec` directly.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
