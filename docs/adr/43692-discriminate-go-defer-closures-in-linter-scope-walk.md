# ADR-43692: Discriminate go/defer Closures in Linter Enclosing-Scope Walk

**Date**: 2026-07-06
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The `execcommandwithoutcontext` linter detects calls to `exec.Command` inside functions that have a `context.Context` parameter, flagging them to use `exec.CommandContext` instead. To find the enclosing context-bearing function, the linter walks up the AST via `cur.Enclosing(...)`. This walk unconditionally crossed every `FuncLit` (anonymous function literal) boundary — including regular callbacks passed to functions like `http.HandleFunc` — treating the outer function's `ctx` as if it were in scope inside those callbacks. This caused false-positive diagnostics and actively harmful autofixes: a server-setup `ctx` was incorrectly attributed to request handlers that should not use it. The same class of bug was previously fixed in the `timesleepnocontext` linter (#42901), but `execcommandwithoutcontext` was not updated at that time.

### Decision

We will break the enclosing-scope walk when the linter encounters a `FuncLit` that is **not** wrapped in a `go` or `defer` statement. For `go func(){}()` and `defer func(){}()` closures, the walk continues because those closures genuinely share the outer function's context lifetime. For all other `FuncLit` nodes (regular callbacks), the walk stops, suppressing the diagnostic. We introduced an `isGoOrDeferClosure` helper — ported from `timesleepnocontext` — that inspects the cursor's grandparent AST node to distinguish the two cases, including parenthesized forms like `defer (func(){})()`.

### Alternatives Considered

#### Alternative 1: Keep Unconditional Enclosing Walk (Status Quo)

The linter would continue to walk through all `FuncLit` boundaries without distinction. This is the pre-fix behavior. It was rejected because it produces false positives for any callback closure nested inside a context-bearing function, leading to incorrect diagnostics and harmful autofixes that inject the wrong `ctx` into request handlers or other independent callback scopes.

#### Alternative 2: Break at ALL FuncLit Boundaries

The walk would stop at every `FuncLit`, regardless of whether it is a `go`/`defer` closure. This prevents all false positives but is overly conservative: it would also suppress valid diagnostics for `defer func() { exec.Command(...) }()` and `go func() { exec.Command(...) }()`, both of which legitimately inherit the outer function's context lifetime and should be flagged. Silencing these would reintroduce real bugs that the linter is designed to catch.

### Consequences

#### Positive
- Eliminates false-positive diagnostics for `exec.Command` calls inside regular callbacks (e.g., `http.HandleFunc`, sync callbacks), preventing incorrect "use exec.CommandContext" suggestions.
- Prevents harmful autofixes that would inject an outer `ctx` — such as a server-setup context — into request handlers or other callback scopes where it is semantically wrong.
- Aligns `execcommandwithoutcontext` behavior with the already-fixed `timesleepnocontext` linter, creating consistent FuncLit boundary semantics across both analyzers.
- New test cases (`GoodHTTPHandleFuncCallbackInCtxFunc`, `GoodSyncCallbackInCtxFunc`, `BadDeferWithCtx`, etc.) provide regression coverage for all four boundary cases.

#### Negative
- The `isGoOrDeferClosure` helper adds a non-trivial AST traversal that future maintainers must understand, including the ParenExpr unwrapping needed for parenthesized `defer`/`go` forms.
- The fix is a heuristic: it classifies closures by their syntactic position (`go`/`defer` grandparent), not by actual context propagation semantics. A callback that manually receives and uses the outer ctx would still be suppressed, potentially creating a false negative (though this is an unusual pattern unlikely to appear in practice).

#### Neutral
- The `isGoOrDeferClosure` function is a direct port from `timesleepnocontext`, so the maintenance burden is shared; any future bug found in one linter should prompt review of the other.
- No API changes; the fix is entirely internal to the linter's AST analysis pass.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
