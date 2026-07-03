# ADR-43126: Enforce Targeted Golint Categories from Lint-Monster Scan

**Date**: 2026-07-03
**Status**: Draft
**Deciders**: Unknown (automated lint enforcement via Copilot SWE agent)

---

### Context

The codebase accumulated 77 custom linter findings from a lint-monster scan across 8 narrow categories. Several categories flag correctness risks: deferred response body closes inside loops cause resource leaks on every iteration; discarded `json.Unmarshal` errors silently hide deserialization failures in tests; blocking `time.Sleep` in the MCP shutdown path ignores context cancellation and can stall graceful teardown. The remaining categories cover idiomatic style (`len(s) > 0` vs `s != ""`, redundant `.Error()` in format verbs, `sort.Slice` vs `slices.SortFunc`, `strings.Split` count anti-pattern). A separate PR tracks the larger `largefunc` backlog. The most consequential change is converting `map[string]bool` set patterns to `map[string]struct{}` in `action_resolver.go` and `action_cache.go`, which propagates through the public API (`GetUsedCacheKeys()`, `PruneOrphanedEntries()`).

### Decision

We will fix all 77 targeted lint findings across 8 categories in a single batch PR rather than deferring to individual feature PRs. We accept the breaking change to `map[string]struct{}` as the canonical set type for cache key tracking. The blocking `time.Sleep` in the MCP shutdown path is replaced with a `select` statement that respects context cancellation. A `probeServer()` helper is extracted from `waitForServerReady` to satisfy both the `deferinloop` and `httprespbodyclose` lint rules simultaneously.

### Alternatives Considered

#### Alternative 1: Fix Findings Incrementally in Feature PRs

Address each lint category only when touching the related file for another purpose, rather than proactively in a dedicated cleanup PR.

Why not chosen: Findings accumulate indefinitely when deferred. The lint-monster scan surfaced this technical debt specifically to be addressed. Deferring creates noise in future feature PR diffs and risks the backlog growing faster than it is resolved.

#### Alternative 2: Suppress Lint Rules with `//nolint:` Directives

Add per-site `//nolint:` comments to silence the 77 findings without changing code semantics.

Why not chosen: Suppression defeats the purpose of the lint rules that flag real correctness issues (discarded errors, deferred closes inside loops, blocking sleeps). Suppressions would mask future regressions of the same class and require ongoing maintenance of the directive list.

### Consequences

#### Positive
- Eliminates 77 known lint findings across 8 categories, reducing CI noise and technical debt.
- `map[string]struct{}` reduces memory overhead for set operations compared to `map[string]bool` (no wasted bool storage per entry).
- Context-aware `select` in the MCP shutdown path allows clean cancellation, preventing the goroutine from blocking for the full shutdown delay when the context is already cancelled.
- Discarded `json.Unmarshal` errors in tests now cause explicit `t.Fatalf` failures rather than silent bad state continuing the test.

#### Negative
- `GetUsedCacheKeys()` and `PruneOrphanedEntries()` have a breaking API change within the package boundary; all call sites must be updated to `map[string]struct{}`.
- The extracted `probeServer()` helper adds a new function to the file surface of `mcp_inspect_mcp_scripts_server.go`, slightly increasing the file's API footprint.

#### Neutral
- 50+ occurrences of `len(s) > 0` replaced with `s != ""` — functionally equivalent, no behavioral change.
- `sort.Slice` replaced with `slices.SortFunc` using `strings.Compare` — semantically equivalent; avoids a name collision with the `cmp` variable present in the same package's test files.
- `strings.Split(...) count` anti-pattern replaced with `strings.Count(...)+1` — algorithmically equivalent, avoids an unnecessary slice allocation.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
