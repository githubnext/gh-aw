# ADR-67818: Share a Single Parameterized Parser for Bounded Skip Conditions

**Date**: 2026-10-11
**Status**: Accepted
**Deciders**: pelikhan (PR author), gh-aw maintainers

---

### Context

The compiler supports two closely related frontmatter triggers, `on.skip-if-match` (bounded by `max`) and `on.skip-if-no-match` (bounded by `min`). In `pkg/workflow/stop_after.go` each was implemented by its own ~140-line extractor, and the two bodies were near-identical: both resolved the `on:` section (preferring the cached `WorkflowData.ParsedFrontmatter.On` over the raw frontmatter map), both accepted either a plain string or an object form, both validated `query`, both coerced the bound from `int`/`int64`/`uint64`/`float64` with a `>= 1` floor, and both extracted an optional `scope`. Duplicated validation logic meant every fix or new accepted type had to be applied twice, and drift between the two code paths was only detectable by reading both functions side by side. The refactor had to be strictly behavior-preserving: the exact contextual error strings (which embed the condition key, bound field name, and an example query) are part of the user-facing compiler UX and are asserted by tests.

### Decision

Both extractors share a single parser, `extractSkipIfBoundFromOn(frontmatter, key, boundField, exampleQuery, workflowData...)`, that returns an internal `skipIfBoundConfig{Query, Bound, Scope}`, with helpers `extractSkipIfBoundObject` and `coerceSkipIfBound`. `extractSkipIfMatchFromOn` and `extractSkipIfNoMatchFromOn` remain as thin typed wrappers that map `Bound` onto `SkipIfMatchConfig.Max` and `SkipIfNoMatchConfig.Min` respectively. The primary driver is maintainability: one implementation of the parsing and validation rules, with the per-condition differences reduced to three string parameters that are interpolated into the existing error messages, so the public config types and all error text stay byte-identical.

The decision is limited to parsing deduplication. Both conditions retain their default bound of 1, accepted input forms, numeric conversions, scope validation, cached-frontmatter precedence, and validation order. Each parser reads only its own bound field; the other condition's bound field remains ignored. Runtime skip comparisons, generated workflow configuration, and the public types are unchanged.

### Alternatives Considered

#### Alternative 1: Keep the two separate extractors as-is

The duplication is self-contained and each function reads linearly without indirection, so no caller or test would need to change. It was rejected because the duplication is the actual defect being addressed (issue #67792): any change to accepted numeric types, bound validation, scope handling, or `on:` caching has to be made and tested twice, and the two copies had already begun to differ only in strings rather than in intent.

#### Alternative 2: Unify on a single shared public config type

Instead of an internal `skipIfBoundConfig` plus wrappers, both triggers could share one exported struct (e.g. `SkipIfConfig{Query, Bound, Scope}`), removing the wrapper layer entirely. This was rejected because `SkipIfMatchConfig.Max` and `SkipIfNoMatchConfig.Min` express different skip thresholds and are consumed by configuration processing and pre-activation step generation in `pkg/workflow/compiler_pre_activation_job.go`, which emits `GH_AW_SKIP_MAX_MATCHES` and `GH_AW_SKIP_MIN_MATCHES` respectively. Replacing the exported types or fields would change the Go API and require changes beyond parsing; it is unnecessary to remove the duplicated implementation. The distinct public types are retained deliberately, not as an interim step toward unification. No public-type unification follow-up is proposed by this ADR.

#### Alternative 3: Generate the two extractors from a template (code generation)

A `go:generate` template could emit both functions from one source, keeping fully specialized code with no runtime parameters. Rejected as disproportionate: two call sites do not justify a generation step, and generated code would be harder to read and debug than a four-argument function.

### Consequences

#### Positive
- Validation, numeric coercion, bound flooring, scope extraction, and cached-`on:` resolution now exist in exactly one place, so fixes apply to both conditions automatically.
- Production parsing duplication is removed in `pkg/workflow/stop_after.go` without changing public types or error text.
- The new table-driven test `pkg/workflow/skip_if_bound_test.go` exercises both conditions through one shared matrix (defaults, bounds, unsupported types, null values, scopes, cached-frontmatter precedence, absent conditions), raising coverage of previously thinly-tested edge cases such as `uint64` overflow and fractional truncation.

#### Negative
- Error messages are now assembled via `fmt.Errorf` format strings with interpolated key/field/example arguments rather than being written out literally, so a wording change is less obvious to review and a mistaken argument order can silently produce a wrong message; correctness now depends on the regression tests that assert exact strings.
- A layer of indirection is introduced: understanding `skip-if-match` parsing requires reading the wrapper, the shared extractor, and two helpers instead of one linear function.
- Pre-existing conversion behavior is preserved — for example converting `^uint64(0)` to `int` yields `-1`, which is rejected by the minimum check, while fractional `float64` values are truncated. Changing these rules requires a separate behavior-change decision.

#### Neutral
- `coerceSkipIfBound` continues to accept only `int`, `int64`, `uint64`, and `float64`; `int32` and other integer widths still produce a type error, matching prior behavior.
- The `exampleQuery` parameter keeps the per-condition example text (`is:issue is:open` vs `is:pr is:open`) in the error output, so documentation-facing strings are unchanged.
- The shared extractor is a package-level function; the unexported typed wrappers remain methods on `*Compiler` for call-site compatibility.
- Custom-linter findings reported by the author in `extractStopAfterFromOn` and `extractSkipIfCheckFailingFromOn` are pre-existing and untouched by this change.

---

*Finalized for PR #67818; records the behavior-preserving parser refactor for issue #67792.*
