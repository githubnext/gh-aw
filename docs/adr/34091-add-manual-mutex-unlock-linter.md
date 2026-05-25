# ADR-34091: Add manual-mutex-unlock Linter

**Date**: 2026-05-22
**Status**: Draft
**Deciders**: Unknown

---

## Part 1 — Narrative (Human-Friendly)

### Context

An automated scan of the codebase (linter-miner run #15) found 12+ instances where `sync.Mutex` / `sync.RWMutex` instances are unlocked manually instead of via `defer mu.Unlock()` — including `pkg/cli/compile_watch.go:189` (with manual unlocks at 204 and 209), multiple sites in `pkg/console/spinner.go:130-168`, and `pkg/cli/docker_images.go:149-151`. Manual unlock is a well-known Go anti-pattern: a panic or early return between `Lock()` and `Unlock()` leaves the mutex held and can deadlock the entire goroutine pool. The repository already houses a family of small, focused, in-house analyzers under `pkg/linters/` (e.g., `fileclosenotdeferred`, `largefunc`, `regexpcompileinfunction`) registered through `cmd/linters/main.go`, so the established convention is to add another analyzer rather than rely on review discipline or external tooling.

### Decision

We will add a new static-analysis linter, `manualmutexunlock`, that flags functions where a `sync.Mutex` or `sync.RWMutex` is unlocked via a non-deferred `Unlock()` / `RUnlock()` call. The linter lives under `pkg/linters/manualmutexunlock/`, is registered in `cmd/linters/main.go` alongside the existing analyzers, walks each `*ast.FuncDecl` body (excluding nested `*ast.FuncLit` closures to avoid false positives) tracking per-variable state keyed by `types.Object` (to correctly handle variable shadowing), and reports the lock position when a variable has a manual `Unlock()` / `RUnlock()` and no matching `defer`. Test files are excluded via the shared `pkg/linters/internal/filecheck.IsTestFile` helper. The implementation mirrors the structure of ADR-33834 (`fileclosenotdeferred`) for consistency.

### Alternatives Considered

#### Alternative 1: Fix the 12+ known instances and rely on review

Patch each flagged call site to use `defer mu.Unlock()` and trust reviewers to catch new instances. Rejected because the same anti-pattern was present in 12+ independent sites across `pkg/cli/` and `pkg/console/`, indicating that human review alone has not been sufficient. A mechanical check on every PR is cheaper than reviewer attention and cannot be forgotten as the codebase grows.

#### Alternative 2: Use a third-party linter (e.g., `gocritic`, `staticcheck`'s SA-rules)

General-purpose Go linters offer mutex-related rules with broader coverage. Rejected to stay consistent with the project's convention of small, focused, in-house analyzers under `pkg/linters/`, each as its own package with custom logic. Pulling in an external linter for a single rule introduces a new dependency surface, inconsistent rule configuration, and noise from rules the project has not opted into.

#### Alternative 3: Combine with a broader "deferred cleanup" linter

Bundle this rule with `fileclosenotdeferred` and any future checks (e.g., `http.Response.Body` close, `context.CancelFunc` invocation) into one analyzer that flags any "resource acquired but cleanup not deferred" pattern. Rejected because the existing convention is one analyzer per rule, which keeps each rule's `Analyzer.Doc` URL narrow, independently disable-able, and easy to extend without coupling unrelated checks. The two linters share a near-identical structural pattern, but coupling them would make it harder to suppress one without the other.

### Consequences

#### Positive
- New non-deferred `Unlock()` / `RUnlock()` patterns introduced after merge are caught by `make golint-custom` and fail in CI rather than landing on `main`.
- The linter follows the same `pkg/linters/<name>/` layout, `Analyzer` shape, and `testdata` convention as the sibling analyzers (notably the sibling `fileclosenotdeferred` from ADR-33834), so contributors can extend it without learning a new pattern.
- Creates incentive to clean up the 12+ pre-existing manual-unlock sites in `pkg/cli/` and `pkg/console/` to maintain a clean linter signal.

#### Negative
- Detection is structural and intentionally narrow: it matches only direct calls `<var>.Lock()` / `<var>.RLock()` / `<var>.Unlock()` / `<var>.RUnlock()` where `<var>`'s static type is `sync.Mutex` / `sync.RWMutex` (or a pointer thereto). It will miss mutexes accessed via field selectors (`s.mu.Lock()`), mutexes returned from helper functions, and locks released through wrapper methods. False negatives are accepted in exchange for low false-positive rate.
- The 12+ pre-existing violations are not fixed in this PR, so the linter cannot be made a blocking CI gate without follow-up work or suppression.
- Adds one more analyzer to the registry, marginally increasing `cmd/linters/main.go` compile and run time.

#### Neutral
- Test files are deliberately excluded via `filecheck.IsTestFile`, matching the convention used by the sibling linters. Test fixtures may legitimately unlock mutexes inline to exercise concurrency paths.
- The diagnostic reports at the lock position, not the manual-unlock position, so the warning points the reader to the place where `defer` should be inserted.
- Re-locking the same variable after a prior unlock is handled by resetting per-variable state on the new `Lock()` call, so cyclical lock/unlock patterns inside a single function are tolerated provided each lock has a matching defer.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Linter Behaviour

1. The analyzer **MUST** be exported as `manualmutexunlock.Analyzer` with `Name` equal to `"manualmutexunlock"`.
2. The analyzer **MUST** report a diagnostic for every local variable whose static type is `sync.Mutex`, `sync.RWMutex`, or a pointer to either, when `Unlock()` or `RUnlock()` is invoked on that variable without a matching `defer` statement in the same `*ast.FuncDecl` body following a `Lock()` or `RLock()` call.
3. The analyzer **MUST NOT** report a diagnostic when the mutex variable's `Unlock()` / `RUnlock()` is invoked via a `defer` statement anywhere in the same function body for the corresponding lock acquisition.
4. The analyzer **MUST NOT** descend into nested `*ast.FuncLit` literals; closures are treated as independent scopes.
5. The analyzer **MUST NOT** report a diagnostic when the containing file is a Go test file as determined by `pkg/linters/internal/filecheck.IsTestFile`.
6. The diagnostic `Pos` **MUST** be the position of the originating `Lock()` / `RLock()` call, not the position of the manual `Unlock()` / `RUnlock()`.
7. The diagnostic `Message` **SHOULD** read `"mutex Unlock() should be deferred immediately after Lock() to prevent deadlocks on panic or early return"` so downstream tooling can match on a stable string.
8. The analyzer **MUST** declare `inspect.Analyzer` in its `Requires` list.
9. Mutex variable identity **MUST** be tracked by `types.Object` (not by identifier name) so that variable shadowing and same-named variables in different scopes are distinguished correctly.

### Registration

1. The analyzer **MUST** be registered in `cmd/linters/main.go` via the `multichecker.Main` argument list alongside the existing custom analyzers.
2. The package import in `cmd/linters/main.go` **MUST** use the path `github.com/github/gh-aw/pkg/linters/manualmutexunlock`.

### Package Layout

1. The linter source **MUST** live under `pkg/linters/manualmutexunlock/`.
2. Test fixtures **MUST** live under `pkg/linters/manualmutexunlock/testdata/src/manualmutexunlock/` and **MUST** use `// want` comments compatible with `golang.org/x/tools/go/analysis/analysistest`.
3. The test fixtures **MUST** include at least one positive case (manual `Unlock()` flagged), one positive case for `sync.RWMutex` (`RUnlock()` flagged), one negative case (`defer mu.Unlock()` not flagged), and one multi-mutex case where one mutex is correctly deferred and another is not.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26311406138) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
