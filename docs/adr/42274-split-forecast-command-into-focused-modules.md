# ADR-42274: Split Forecast Command into Focused Modules

**Date**: 2026-06-29
**Status**: Draft
**Deciders**: Unknown

---

### Context

`pkg/cli/forecast.go` had grown to approximately 1,440 lines, mixing five distinct concerns in a single file: command orchestration, GitHub API workflow resolution with rate-limit retry logic, Markdown frontmatter metadata extraction, run sampling and AIC computation (including Monte Carlo simulation), and JSON/table rendering with format helpers. This made it difficult for contributors to locate, understand, or modify any single concern without reading through the entire file. The `pkg/cli` package follows a convention of grouping command logic by command name, but no convention existed for splitting a single command across multiple files by domain.

### Decision

We will decompose `pkg/cli/forecast.go` into five domain-focused files within the same `pkg/cli` package — `forecast_types.go`, `forecast_resolution.go`, `forecast_metadata.go`, `forecast_compute.go`, and `forecast_render.go` — while keeping `forecast.go` as a thin orchestration entrypoint containing only `RunForecast` and `normalizeForecastRunError`. All existing types, function signatures, and public API behaviour are preserved unchanged; the refactor is purely structural. Test placement is aligned with implementation by moving format-helper tests to `forecast_render_test.go`.

### Alternatives Considered

#### Alternative 1: Keep the monolithic file (status quo)

Continue adding to `forecast.go` as new features land. This avoids the file-navigation cost of the refactor and keeps git blame and history for all forecast logic in one place. It was rejected because the file had already become unwieldy at 1,440 lines, and each new feature (eval mode, variant fractions, per-run samples) compounds the problem. The cost of deferral grows with each addition.

#### Alternative 2: Extract to a dedicated sub-package (`pkg/cli/forecast/`)

Create a `pkg/cli/forecast` sub-package with separate packages for types, compute, resolution, metadata, and render. This would enforce encapsulation via Go package boundaries, making dependencies explicit. It was not chosen for this refactor because it would require exporting all shared types and internal helpers, significantly widening the API surface and increasing the scope of the change. The file-split approach achieves the readability goal with minimal structural impact, leaving the package-extraction option open for a future step if inter-domain coupling needs to be enforced.

### Consequences

#### Positive
- Each file has a single domain responsibility, making it straightforward to locate any part of the forecast command's logic.
- Test files align with implementation files (`forecast_render_test.go` alongside `forecast_render.go`), making it easier to find tests for a given module.
- The thin `forecast.go` entrypoint clearly communicates the command's top-level structure without scrolling through implementation details.

#### Negative
- Five additional files are introduced for one logical command, increasing the number of files a contributor must be aware of.
- No interface boundaries are introduced between modules; function-variable indirection (e.g. `forecastLoadCachedRunAIC`, `forecastDownloadRunArtifacts`) remains in package-level `var` blocks, so coupling between modules is still implicit rather than enforced by the type system.

#### Neutral
- All symbols remain in the `cli` package; no import paths change, and callers of `RunForecast` are unaffected.
- The refactor is a pure move with no logic changes, so existing tests continue to exercise the same code paths without modification.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
