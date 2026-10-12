# ADR-67818: Share a Single Parameterized Parser for Bounded Skip Conditions

**Date**: 2026-02-18
**Status**: Draft
**Deciders**: pelikhan (PR author), gh-aw maintainers

---

### Context

The compiler supports two closely related frontmatter triggers, `on.skip-if-match` (bounded by `max`) and `on.skip-if-no-match` (bounded by `min`). In `pkg/workflow/stop_after.go` each was implemented by its own ~140-line extractor, and the two bodies were near-identical: both resolved the `on:` section (preferring the cached `WorkflowData.ParsedFrontmatter.On` over the raw frontmatter map), both accepted either a plain string or an object form, both validated `query`, both coerced the bound from `int`/`int64`/`uint64`/`float64` with a `>= 1` floor, and both extracted an optional `scope`. Duplicated validation logic meant every fix or new accepted type had to be applied twice, and drift between the two code paths was only detectable by reading both functions side by side. The refactor had to be strictly behavior-preserving: the exact contextual error strings (which embed the condition key, bound field name, and an example query) are part of the user-facing compiler UX and are asserted by tests.

### Decision

We will collapse both extractors into a single unduplicated parser, `extractSkipIfBoundFromOn(frontmatter, key, boundField, exampleQuery, workflowData...)`, that returns an internal `skipIfBoundConfig{Query, Bound, Scope}`, with helpers `extractSkipIfBoundObject` and `coerceSkipIfBound`. `extractSkipIfMatchFromOn` and `extractSkipIfNoMatchFromOn` remain as thin typed wrappers that map `Bound` onto `SkipIfMatchConfig.Max` and `SkipIfNoMatchConfig.Min` respectively. The primary driver is maintainability: one implementation of the parsing and validation rules, with the per-condition differences reduced to three string parameters that are interpolated into the existing error messages, so the public config types and all error text stay byte-identical.

### Alternatives Considered

#### Alternative 1: Keep the two separate extractors as-is

The duplication is self-contained and each function reads linearly without indirection, so no caller or test would need to change. It was rejected because the duplication is the actual defect being addressed (issue #67792): any change to accepted numeric types, bound validation, scope handling, or `on:` caching has to be made and tested twice, and the two copies had already begun to differ only in strings rather than in intent.

#### Alternative 2: Unify on a single shared public config type

Instead of an internal `skipIfBoundConfig` plus wrappers, both triggers could share one exported struct (e.g. `SkipIfConfig{Query, Bound, Scope}`), removing the wrapper layer entirely. This was rejected because `SkipIfMatchConfig.Max` and `SkipIfNoMatchConfig.Min` are used across the compiler and lock-file generation; renaming them would turn a local cleanup into a cross-cutting, non-behavior-preserving change. The wrapper approach keeps the blast radius inside one file. *[TODO: verify whether a follow-up should still unify the public types.]*

#### Alternative 3: Generate the two extractors from a template (code generation)

A `go:generate` template could emit both functions from one source, keeping fully specialized code with no runtime parameters. Rejected as disproportionate: two call sites do not justify a generation step, and generated code would be harder to read and debug than a four-argument function.

### Consequences

#### Positive
- Validation, numeric coercion, bound flooring, scope extraction, and cached-`on:` resolution now exist in exactly one place, so fixes apply to both conditions automatically.
- Net removal of ~64 lines of production code in `pkg/workflow/stop_after.go` (77 added, 141 deleted) with no change to public types or error text.
- The new table-driven test `pkg/workflow/skip_if_bound_test.go` exercises both conditions through one shared matrix (defaults, bounds, unsupported types, null values, scopes, cached-frontmatter precedence, absent conditions), raising coverage of previously thinly-tested edge cases such as `uint64` overflow and fractional truncation.

#### Negative
- Error messages are now assembled via `fmt.Errorf` format strings with interpolated key/field/example arguments rather than being written out literally, so a wording change is less obvious to review and a mistaken argument order can silently produce a wrong message; correctness now depends on the regression tests that assert exact strings.
- A layer of indirection is introduced: understanding `skip-if-match` parsing requires reading the wrapper, the shared extractor, and two helpers instead of one linear function.
- Pre-existing quirks are now shared by construction — for example `uint64` overflow coercing to `-1` and `float64` truncation — so they are locked in for both conditions unless deliberately changed.

#### Neutral
- `coerceSkipIfBound` continues to accept only `int`, `int64`, `uint64`, and `float64`; `int32` and other integer widths still produce a type error, matching prior behavior.
- The `exampleQuery` parameter keeps the per-condition example text (`is:issue is:open` vs `is:pr is:open`) in the error output, so documentation-facing strings are unchanged.
- The `Compiler` receiver is no longer used by the shared extractor, which is now a package-level function; the exported wrappers keep their methods on `*Compiler` for call-site compatibility.
- Custom-linter findings reported by the author in `extractStopAfterFromOn` and `extractSkipIfCheckFailingFromOn` are pre-existing and untouched by this change.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
