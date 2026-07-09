# ADR-44461: Adopt Idiomatic Go Patterns for Custom Linter Compliance

**Date**: 2026-07-09
**Status**: Draft
**Deciders**: Unknown (automated PR by Copilot SWE Agent)

---

### Context

The repository maintains a bespoke `make golint-custom` suite of `go/analysis` linters (see `pkg/linters/`) that enforce project-wide Go idioms. Over time, the codebase accumulated ~128 new lines of business logic that contained patterns flagged by these linters. The accumulated diagnostics fell into five categories across `pkg/cli`, `pkg/workflow`, `pkg/parser`, and `pkg/logger`: (1) allocating `[]byte`→`string` conversions for equality (`bytes.Equal` is already covered by ADR-44389); (2) `map[string]bool` used as membership sets where `map[string]struct{}` avoids per-entry boolean allocations; (3) `h.Write([]byte(s))` hash writes that allocate unnecessarily when `io.WriteString` can be used if the writer implements `io.StringWriter`; (4) `len(s) > 0` empty-string checks where the idiomatic form is `s != ""`; and (5) bare `time.Sleep` calls that ignore context cancellation. Each category has an established preferred idiom in Go and a registered linter diagnostic. The diagnostics form ongoing noise in the CI lint step and are tracked separately from the function-length (`largefunc`) backlog.

### Decision

We will apply all non-`largefunc` custom linter findings in a single compliance pass across the affected packages: replace `map[string]bool` membership sets with `map[string]struct{}` (including API and test updates), switch hash writer calls from `h.Write([]byte(s))` to `io.WriteString(h, s)`, replace `len(s) > 0` guards with `s != ""` at ~50 call sites, replace bare `time.Sleep` with a context-aware `select { case <-time.After(d): case <-ctx.Done(): }` pattern in `mcp_inspect.go`, and replace untyped `sort.Slice` with type-safe `slices.SortFunc`. Discarded `json.Unmarshal` errors in test files are also handled. The pass brings `make golint-custom` output to zero non-`largefunc` diagnostics.

### Alternatives Considered

#### Alternative 1: Suppress or Disable the Linter Rules

Linter rules could be suppressed per-file (`//nolint:lintname`) or removed from the custom suite, treating the flagged patterns as acceptable. This was rejected because the patterns are flagged for real reasons (unnecessary allocations, non-idiomatic style, context-ignoring sleeps), and suppressing them would permanently hide future regressions of the same patterns in new code.

#### Alternative 2: Fix One Category at a Time Across Separate PRs

Each pattern category could be addressed in its own PR, allowing focused review. This was rejected in favor of a single consolidated pass because (a) the categories are mechanically similar and share the same motivation (linter compliance), (b) splitting them would produce more merge conflicts and more CI runs for equivalent net value, and (c) the individual changes are small enough that a single PR remains reviewable.

### Consequences

#### Positive
- `make golint-custom` produces zero non-`largefunc` diagnostics after this PR, eliminating CI noise and providing a clean baseline.
- `map[string]struct{}` sets avoid storing a redundant `bool` value per entry; `io.WriteString` avoids a `[]byte` heap allocation per hash write call.
- The context-aware shutdown in `mcp_inspect.go` correctly respects cancellation instead of blocking the goroutine for the full sleep duration.
- Typed `slices.SortFunc` surfaces type errors at compile time that `sort.Slice` would not catch.

#### Negative
- `map[string]struct{}` is more syntactically verbose than `map[string]bool`; set membership checks require the two-value `_, ok := m[k]` idiom instead of the direct boolean `m[k]`.
- The breadth of the change (57 files, ~128 additions) increases the review surface and raises the probability of a rebase conflict if another branch touches the same files.

#### Neutral
- The `map[string]struct{}` API change in `action_resolver.go` and `action_cache.go` is breaking at the Go type level; all internal callers and tests are updated in the same PR.
- `io.WriteString` falls back to a plain `Write([]byte(s))` call if the writer does not implement `io.StringWriter`; behavior is identical, only the allocation path differs.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
