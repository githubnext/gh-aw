# ADR-35613: Consolidate Sanitization Helpers and Unify Compiler Error Formatter as an Options-Based API

**Date**: 2026-05-29
**Status**: Draft
**Deciders**: Unknown

---

## Part 1 — Narrative (Human-Friendly)

### Context

The codebase had grown three independent duplicate implementations of filename-style sanitization (`sanitizePath` in `pkg/parser/import_cache.go`, `sanitizeRepoPath` in `pkg/cli/update_command.go` and `pkg/cli/deploy_command.go`, and the package-level `stringutil.SanitizeForFilename`), each with subtly different separators (`_`, `__`, `-`). In parallel, the workflow compiler exposed three overloads for emitting formatted diagnostics — `formatCompilerError`, `formatCompilerErrorWithPosition`, and `formatCompilerErrorWithContext` — and each new piece of diagnostic metadata required adding yet another overload. A third inconsistency was that some callers reached `stringutil.SanitizeName` directly while others routed through the `pkg/workflow` re-export layer (`workflow.SanitizeName` / `workflow.SanitizeOptions`), making the intended layering ambiguous. The combined effect was duplicate logic, drift risk in sanitization, and an overload family that was about to grow further.

### Decision

We will consolidate the sanitization surface and the compiler-error formatter surface as follows:

1. Add `stringutil.SanitizeForFilenameWithSeparator(slug, separator string)` as the single configurable filename-sanitization helper, and re-implement `stringutil.SanitizeForFilename` as a thin `"-"`-separator wrapper over it. Remove the duplicate `sanitizePath` and `sanitizeRepoPath` helpers and route every caller through the shared helper.
2. Collapse `formatCompilerError`, `formatCompilerErrorWithPosition`, and `formatCompilerErrorWithContext` into a single function `formatCompilerError(opts compilerErrorOpts)` where `compilerErrorOpts` carries `FilePath`, `Line`, `Column`, `ErrType`, `Message`, `Cause`, and `Context`. Defaults (`Line=1`, `Column=1`) and `WorkflowValidationError` location promotion are applied inside this single function.
3. Keep the `pkg/workflow` re-export layer (`SanitizeOptions` / `SanitizeName`) as the canonical entry point for workflow-package callers, and route the previously bypassing CLI caller (`pkg/cli/logs_run_processor.go`) through `workflow.SanitizeName` to enforce that layering.

### Alternatives Considered

#### Alternative 1: Add a fourth overload (`formatCompilerErrorWithContextAndCause`)

Continue the existing pattern of adding a new overload each time the diagnostic carries an additional optional field. Rejected because the overload family was already at three with overlapping responsibilities, and each new metadata field (context, cause, structured validation info) would force a combinatorial expansion of overloads and call sites. The team has already had to add `formatCompilerErrorWithContext` once; adding a fourth would entrench the pattern.

#### Alternative 2: Functional options pattern (`formatCompilerError(opts ...Option)`)

Use Go's idiomatic functional-options pattern with helpers like `WithLine(n)`, `WithContext(lines)`, etc. Rejected as heavier weight than needed: the diagnostic shape is small, the option set is closed, and the struct literal at the call site is at least as readable as a chain of `WithX(...)` calls while requiring no per-field helper functions. A struct also lets the `WorkflowValidationError`-promotion logic mutate one struct field in place rather than threading through an option pipeline.

#### Alternative 3: Move `SanitizeName` fully into `stringutil` and delete the `workflow` re-exports

Delete `workflow.SanitizeOptions` / `workflow.SanitizeName` and require all callers to import `stringutil` directly. Rejected because the re-export layer documents an intentional boundary — workflow-facing callers should depend on `pkg/workflow`, not reach across into `pkg/stringutil`. Deleting the re-export would push that import-graph concern onto every workflow consumer and make the layering harder to enforce by review. The chosen direction documents the re-export's intent in `pkg/workflow/strings.go` and migrates the one bypassing caller.

#### Alternative 4: Per-caller bespoke sanitizers

Leave each caller with its own small sanitizer (the status quo). Rejected because the separators were already drifting (`_` vs `__` vs `-`) and the only thing distinguishing the helpers was the separator string — a parameter, not a behavior.

### Consequences

#### Positive
- Single source of truth for repository-slug / path filename sanitization, parameterized only by separator.
- A single, extensible compiler-error formatter: adding a new diagnostic field is a struct-field addition, not a new function.
- The `pkg/workflow` re-export layer is now the only sanctioned path for workflow-package sanitization, making layering reviewable by import inspection.
- `WorkflowValidationError` position promotion is implemented exactly once, instead of risking drift across overloads.

#### Negative
- Call sites are more verbose: `formatCompilerError(compilerErrorOpts{FilePath: x, ErrType: "error", Message: m, Cause: e})` is longer than the previous positional form, and many call sites had to be rewritten in this PR.
- The struct-based API loses compile-time enforcement that `Message` is supplied; a caller could construct `compilerErrorOpts{}` and emit an empty diagnostic.
- `SanitizeForFilenameWithSeparator` silently falls back to `"-"` when given an empty separator, which is convenient but hides a likely bug at the call site.

#### Neutral
- Introduces two small internal helpers (`isFilenameSafeRune`, `normalizePosition`) that did not previously exist.
- The PR touches a large number of unrelated workflow validation files purely to update call sites; this is mechanical churn, not behavior change.
- `pkg/cli/logs_run_processor.go` now imports `pkg/workflow` in addition to `pkg/stringutil` because of the re-export routing decision.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Filename Sanitization

1. New code that converts a repository slug or path to a filename-safe string **MUST** use `stringutil.SanitizeForFilenameWithSeparator(slug, separator)` or its `"-"`-separator wrapper `stringutil.SanitizeForFilename(slug)`.
2. Packages other than `pkg/stringutil` **MUST NOT** define local helpers that replicate filename sanitization (for example, helpers named `sanitizePath`, `sanitizeRepoPath`, or equivalents).
3. Callers that previously relied on a non-`"-"` separator (for example `"_"` for cache paths, `"__"` for repo checkout directories) **MUST** pass that separator explicitly to `SanitizeForFilenameWithSeparator` rather than reintroducing a local sanitizer.
4. `SanitizeForFilenameWithSeparator` **SHOULD** treat an empty `separator` argument as a defaulting hint to `"-"`; callers **SHOULD NOT** rely on this fallback and **SHOULD** pass an explicit separator.

### Compiler Error Formatting

1. Code in `pkg/workflow` that emits a console-formatted compiler diagnostic **MUST** use `formatCompilerError(compilerErrorOpts{...})`.
2. Code in `pkg/workflow` **MUST NOT** reintroduce overloads such as `formatCompilerErrorWithPosition` or `formatCompilerErrorWithContext`. Additional diagnostic metadata **MUST** be added as new fields on `compilerErrorOpts`.
3. Callers **MUST** populate at minimum `FilePath`, `ErrType`, and `Message` on `compilerErrorOpts`. Callers **MAY** omit `Line` and `Column`, in which case both default to `1`.
4. Callers **MUST NOT** pre-promote `WorkflowValidationError` position/file into `compilerErrorOpts.Line` / `Column` / `FilePath`; this promotion is the responsibility of `formatCompilerError` itself.
5. Callers that have source-context lines available **SHOULD** populate `compilerErrorOpts.Context` so that Rust-style snippet rendering is used.
6. Code that needs to detect whether an `error` is already a console-formatted compiler diagnostic **MUST** call `isFormattedCompilerError(err)` and **MUST NOT** rely on string-contains checks against the formatted text.

### SanitizeName Layering

1. Callers within `pkg/workflow` and its CLI consumers **MUST** use `workflow.SanitizeName` and `workflow.SanitizeOptions` rather than `stringutil.SanitizeName` / `stringutil.SanitizeOptions` directly.
2. Code in `pkg/stringutil` **MUST** remain the canonical implementation of `SanitizeName`; `pkg/workflow/strings.go` **MUST** continue to be a thin re-export.
3. New non-workflow packages that need name sanitization **MAY** call `stringutil.SanitizeName` directly; the re-export rule applies to workflow-facing callers only.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26617861244) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
