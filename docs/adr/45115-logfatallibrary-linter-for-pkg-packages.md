# ADR-45115: Add logfatallibrary Linter to Flag log.Fatal Calls in Library Packages

**Date**: 2026-07-13
**Status**: Draft
**Deciders**: Unknown

---

### Context

The repository enforces a culture of safe library code through static analysis linters. Library packages under `pkg/` must not terminate the process directly because callers have no opportunity to recover or clean up. The existing `osexitinlibrary` linter catches direct `os.Exit` calls, and `panicinlibrarycode` catches `panic`. However, `log.Fatal`, `log.Fatalf`, and `log.Fatalln` from the standard `log` package also call `os.Exit(1)` internally, making them equally dangerous in library code. This implicit exit path bypasses deferred cleanup (open files, database connections, in-flight requests), makes packages untestable in isolation (a test binary can be aborted by a library call), and prevents callers from handling errors gracefully. No automated check existed to catch this pattern.

### Decision

We will add a new `logfatallibrary` Go analysis pass under `pkg/linters/logfatallibrary/` that reports any call to `log.Fatal`, `log.Fatalf`, or `log.Fatalln` inside a non-`cmd/` package. Entry-point packages (those under `cmd/` or with path suffix `/main`) are explicitly exempted because process termination is acceptable in executables. Library code must return errors instead, consistent with standard Go idiom and the existing linter family.

### Alternatives Considered

#### Alternative 1: Extend the existing `osexitinlibrary` linter

The existing linter already detects direct `os.Exit` calls in library packages. It could be extended to also detect `log.Fatal*` calls, consolidating the logic in one place. This was rejected because the two linters flag conceptually distinct constructs (explicit vs. implicit exit) and keeping them separate makes each linter's doc string, suppression directive name, and test fixture self-contained. Merging them would also widen the blast radius of any future change to either rule.

#### Alternative 2: Rely on documentation and code review

The prohibition on `log.Fatal*` in library code could be documented in the contributing guide and enforced through code review alone. This was rejected because code review is inconsistent and does not scale — the pattern is easy to miss, especially for contributors unfamiliar with the implicit `os.Exit` behavior of `log.Fatal`. Automated enforcement is the only approach that catches every occurrence unconditionally.

### Consequences

#### Positive
- Every future PR that introduces `log.Fatal*` in a library package is flagged automatically, eliminating a class of hidden process-termination bugs.
- The new linter is consistent with the existing `osexitinlibrary` and `panicinlibrarycode` passes, extending the suite without introducing new patterns or conventions.

#### Negative
- Existing library code that currently uses `log.Fatal*` must be migrated to return errors, which may require changes to function signatures and call sites.
- Developers who are unaware that `log.Fatal` calls `os.Exit` may be surprised by the diagnostic and need to understand the reason before they can fix it.

#### Neutral
- The linter supports `//nolint:logfatallibrary` directives (both same-line and previous-line) for the rare legitimate suppression case.
- Test files (files matching `_test.go`) and test packages (path suffix `.test`) are excluded from analysis, so test helpers that use `log.Fatal` for test setup are unaffected.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
