# ADR-51573: Coverage-Aware Gating for Performance-Oriented Custom Linters

**Date**: 2026-08-09
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The repository maintains a suite of custom Go static-analysis linters. A subset of these linters flag micro-optimizations that only matter on hot paths — code that executes frequently under real workloads. Examples include `stringsconcatloop` (O(n²) string concatenation), `appendbytestring`, `bytesbufferstring`, `seenmapbool`, and 9 others. Applying these rules uniformly to all code — including dead code or paths that tests never reach — generates review noise without a measurable performance payoff. At the same time, purely stylistic linters (e.g. `stringsindexcontains`, `lenstringzero`) carry no performance benefit regardless of execution frequency, so gating them on coverage would be misleading.

### Decision

We will add a shared `pkg/linters/internal/coverage` package that loads a Go coverage profile (produced by `go test -covermode=count -coverprofile=<path>`, referenced via the `GH_AW_LINT_COVERAGE_PROFILE` environment variable) and exposes two helpers: `ShouldApply(pass, pos, threshold)` to gate a diagnostic on the line's recorded execution hit count, and `RegisterHotThresholdFlag(analyzer)` to register a per-linter `-hot-threshold` flag. All 13 allocation/perf-oriented linters will integrate this mechanism via an `init()` function to avoid analyzer-initialization cycles. When no profile is loaded (the default), gating is a no-op and all linters behave exactly as before; purely stylistic linters are left ungated.

### Alternatives Considered

#### Alternative 1: Uniform application (status quo)

Continue running all linters on all code regardless of coverage. This is simpler — no new package, no env-var convention, no per-linter flag — but produces diagnostic noise on dead or rarely-executed code paths, leading to review churn with no measurable performance return.

#### Alternative 2: Per-site `nolint` suppression

Require developers to suppress false-positive perf diagnostics on a case-by-case basis with `//nolint:stringsconcatloop` (or similar) comments. This keeps the linter infrastructure simple but places the burden on individual contributors each time a new cold-path finding appears, and does not scale as the codebase or linter set grows.

### Consequences

#### Positive
- Perf linters fire only on code paths that tests actually exercise, eliminating diagnostic noise on dead or rarely-executed code.
- Fully permissive fallback: existing CI pipelines that do not set `GH_AW_LINT_COVERAGE_PROFILE` see no behavioral change.
- The `-hot-threshold` flag gives per-linter control; passing `0` disables coverage gating for a specific linter even when a profile is present.
- A documented, reusable pattern (`init()` + `coverage.RegisterHotThresholdFlag` + `coverage.ShouldApply`) makes it straightforward to wire future perf linters into the same mechanism.

#### Negative
- Activating coverage gating requires generating a coverage profile (`go test -covermode=count`) and setting `GH_AW_LINT_COVERAGE_PROFILE` in the lint environment; CI/CD pipeline configuration changes may be needed to realize the benefit.
- Two behavioral modes (gated vs. ungated) exist per perf linter, increasing the number of states to reason about when debugging a linter that is unexpectedly silent.

#### Neutral
- Purely stylistic/readability linters (`stringsindexcontains`, `stringsindexhasprefix`, `stringscountcontains`, `lenstringzero`, etc.) are intentionally excluded from coverage gating, which means the two linter categories now diverge in their configuration surface.
- The `init()` pattern (rather than a `var` initializer) is required to avoid an `Analyzer`/`run`/flag initialization cycle; this convention is documented in `pkg/linters/README.md` and `.github/skills/go-linters/SKILL.md`.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
