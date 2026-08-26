# ADR-56058: Consolidate Duplicated String-or-Slice Coercion into typeutil.NormalizeStringSlice

**Date**: 2026-08-26
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

GitHub Actions YAML allows many fields (e.g. `needs`, `on`, `runs-on`) to be expressed as either a scalar string or a list of strings. This means Go decoding produces values typed as `string`, `[]string`, or `[]any` for the same logical field. Across `pkg/workflow` and `pkg/cli`, roughly ten separate functions independently implemented the same `string | []any | []string → []string` type switch — `normalizeStringOrStringSlice`, `parseRepoMemoryStringList`, `anySliceToStrings`, `jobNeeds`, and others — each with subtly drifting behavior around empty-string filtering, sorting, deduplication, and mutation safety. The `typeutil` package already housed analogous shared helpers for numeric coercion (`ParseIntValue`), establishing a precedent for this kind of canonical primitive.

### Decision

We will add `typeutil.NormalizeStringSlice(v any) []string` as the single canonical implementation of `string | []any | []string → []string` coercion and migrate all existing near-identical helpers to call it. The function returns a copy for `[]string` inputs (mutation safety), keeps only string-typed elements from `[]any`, wraps a scalar `string` in a single-element slice, and returns `nil` for unsupported types — matching the lenient convention of the other `typeutil` helpers. Domain-specific post-processing (trimming, empty-string filtering, deduplication, sorting) is intentionally left to each call site, since that is where the previous implementations diverged.

### Alternatives Considered

#### Alternative 1: Keep per-package helpers unchanged

Each package retains its own type-switch implementation and is responsible for its own consistency. This requires no code movement and avoids any cross-package dependency change. Rejected because the behavioral drift between implementations was already observable (e.g. `mutableStringSlice` used `fmt.Sprint` to coerce non-string `[]any` elements while all other helpers silently dropped them), and any future spec change or bug fix would need to be applied to ~10 separate locations independently.

#### Alternative 2: Add the helper to `stringutil` instead of `typeutil`

The function could live in the existing `stringutil` package rather than `typeutil`, grouping it with other string utilities. Rejected because the function's primary role is *type coercion* (converting an `any` to `[]string`) rather than string manipulation, and `typeutil` already contains the established pattern of lenient type coercions (`ParseIntValue`, `LookupString`). Placing it in `stringutil` would blur the boundary between type adaptation and string processing.

### Consequences

#### Positive
- Single source of truth for the coercion logic — a bug fix or spec change is made in one place.
- `[]string` inputs are now copied rather than aliased, so callers that mutate the returned slice can no longer accidentally corrupt the source.
- Behavioral consistency across all migrated call sites: non-string elements in `[]any` are uniformly skipped rather than handled inconsistently.
- Coverage is consolidated into `pkg/typeutil/stringslice_test.go`, removing redundant test duplication across packages.

#### Negative
- One intentional behavioral change in `mutableStringSlice` (`pkg/cli/outcome_eval_update.go`): the previous `[]any` branch used `fmt.Sprint(value)` to stringify non-string elements (e.g. integers), so they were included in the output after trimming. The new helper skips non-string elements entirely. This only affects state comparison of list fields such as issue labels; in practice such fields should only contain string values, but the change may silently drop values where the previous behavior preserved them.
- `parseStringSliceAny` is retained as a wrapper rather than eliminated: it adds debug logging on skipped elements that the canonical helper does not, so two related abstractions now coexist. Future contributors must understand when to use each.

#### Neutral
- The `typeutil` package acquires a new `stringslice.go` file alongside its existing `convert.go`, `lookup.go`, and `allocation.go`.
- Two unit tests (`TestJobNeeds`, `TestNormalizeStringOrStringSlice`) are removed from their original packages; equivalent coverage lives in `pkg/typeutil/stringslice_test.go`.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
