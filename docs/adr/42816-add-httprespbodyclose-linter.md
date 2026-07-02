# ADR-42816: Add httprespbodyclose Custom Go Linter

**Date**: 2026-07-01
**Status**: Accepted
**Deciders**: gh-aw maintainers, @copilot

---

### Context

Go code that makes HTTP requests must close the response body to release the underlying TCP connection. A common mistake is calling `resp.Body.Close()` directly (without `defer`), which causes a resource leak if the function returns early before the close call is reached — for example, when an error occurs mid-function. The `pkg/` and `cmd/` directories in this repository contain several files that perform HTTP calls (`pkg/parser/remote_fetch.go`, `pkg/cli/import_url_fetcher.go`), making this a real risk. Existing linters in the suite do not cover this pattern: `fileclosenotdeferred` handles `os.Open`/`os.Create`/`os.OpenFile` only, and `httpnoctx` flags context-free HTTP calls but not body-close patterns.

### Decision

We will add a new custom Go static analysis pass, `httprespbodyclose`, to the gh-aw linter suite. The analyzer uses Go type information (`*net/http.Response`) to identify HTTP response variables and flags any `resp.Body.Close()` call that is not guarded by `defer`. It is registered in `cmd/linters/main.go` alongside existing custom analyzers and supports suppression via `//nolint:httprespbodyclose` directives.

### Alternatives Considered

#### Alternative 1: Use an existing third-party linter (e.g., `timakin/bodyclose`)

The open-source `bodyclose` linter covers the same anti-pattern. We chose not to adopt it because the gh-aw suite uses a custom multichecker architecture with shared internal utilities (`astutil`, `filecheck`, `nolint`) that provide consistent behavior (test-file exclusion, nolint suppression, position reporting). Importing a third-party linter would bypass this infrastructure and introduce an external dependency subject to upstream changes.

#### Alternative 2: Extend `fileclosenotdeferred` to cover HTTP response bodies

The `fileclosenotdeferred` linter already handles deferred-close enforcement for file handles. Extending it to also cover `*http.Response` bodies was considered to reduce the number of separate analyzer packages. This was rejected because `*http.Response` has different semantics (it is not obtained via `os.Open`-family calls, and the type check requires `net/http` type resolution), which would make the combined linter harder to reason about and test independently.

### Consequences

#### Positive
- HTTP response body leaks are caught at compile time, before they reach production.
- The analyzer is type-aware (`*net/http.Response` via `go/types`), which reduces false positives compared to purely syntactic string-matching approaches.
- Follows the established pattern for all custom linters in this suite, making it easy for contributors to understand and extend.

#### Negative
- Developers writing HTTP code must now be aware of the linter and may occasionally need to add `//nolint:httprespbodyclose` for intentional non-deferred closes (e.g., when the response is returned to the caller).
- The linter does not track `*http.Response` across function boundaries: if a response is passed to a helper that closes it, the analyzer cannot determine that and may produce false negatives.

#### Neutral
- The linter skips test files by convention (via `filecheck.IsTestFile`), consistent with other analyzers in the suite.
- A new package directory (`pkg/linters/httprespbodyclose/`) is added, contributing to the total package count of the linter suite.
