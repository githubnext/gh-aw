# ADR-44097: Extend envutil with Typed Helpers for Boolean and String Environment Variables

**Date**: 2026-07-08
**Status**: Draft
**Deciders**: Unknown

---

### Context

The codebase has a convention of reading environment variables through `pkg/envutil.GetIntFromEnv`, which centralizes defaulting, bounds-validation, and debug logging for integer-typed env vars. However, boolean and string env vars in `pkg/cli` and `pkg/parser` were still read via direct `os.Getenv` calls, suppressed with `//nolint:osgetenvlibrary` annotations. This produced inconsistent behavior: CI detection was replicated inline across three call sites in `pkg/cli` (`os.Getenv("CI") != ""`), `GO_TEST_MODE` was compared as a raw string rather than using Go's canonical boolean parsing, and `CODESPACES` used `strings.EqualFold` on a raw string rather than a shared accessor. Sensitive tokens (`GITHUB_TOKEN`, `GH_TOKEN`) were read directly without debug-safe logging. The growing number of exemptions signalled that the existing helper surface was incomplete for the types of env vars the codebase actually uses.

### Decision

We will add `GetBoolFromEnv` and `GetStringFromEnv` helpers to `pkg/envutil`, consistent in signature and behavior with the existing `GetIntFromEnv`, and migrate all direct `os.Getenv` calls in `pkg/cli` and `pkg/parser` to use these helpers or the existing `IsRunningInCI()` / `isRunningInCodespace()` wrapper functions that now delegate to them. The shared internal `warn` function extracted in this PR eliminates duplicated warning routing code across all three typed helpers.

### Alternatives Considered

#### Alternative 1: Retain Direct `os.Getenv` with Nolint Suppressions

Continue using raw `os.Getenv` at each call site with `//nolint:osgetenvlibrary` to satisfy the linter. Each call site would keep its own string-comparison, defaulting, and logging logic. This avoids introducing a new abstraction, but perpetuates inconsistency: `GO_TEST_MODE == "true"` rejects `"1"`, `"TRUE"`, and `"yes"` as truthy, while the rest of the codebase would gain consistent boolean parsing via `strconv.ParseBool`. It also makes debug-safe logging of sensitive env values (never log the value itself) opt-in per call site rather than the default.

#### Alternative 2: Dependency-Injection of an Environment Reader Interface

Introduce an `EnvReader` interface and inject it into all consumers, allowing env access to be mocked in tests without `os.Setenv`. This would improve test isolation for packages that currently must manipulate real process environment state. However, it is a significantly larger change, requires threading the interface through many function signatures, and conflicts with the existing `envutil` "static helper" design that callers already adopt. The present PR's test coverage uses `os.Setenv`/`os.Unsetenv` with deferred cleanup, which is adequate for the current test surface.

### Consequences

#### Positive
- `GetBoolFromEnv` accepts all spellings recognised by `strconv.ParseBool` (`"1"`, `"t"`, `"TRUE"`, etc.), making boolean env var handling consistent with Go conventions across the codebase.
- `GetStringFromEnv` logs `Using ENV_VAR from environment` rather than the value itself, making it safe to use with secret-like variables (`GITHUB_TOKEN`, `GH_TOKEN`) without risk of echoing credentials in debug output.
- Removing `//nolint:osgetenvlibrary` suppressions from `pkg/cli` and `pkg/parser` means the linter now enforces the centralized pattern without human-reviewed exceptions.
- The shared `warn` helper in `envutil.go` eliminates a duplicated closure that was previously inlined inside `GetIntFromEnv`, reducing surface area for divergence.

#### Negative
- `GetStringFromEnv` treats an empty string value identically to an unset variable and returns `defaultValue` in both cases. Callers that need to distinguish between `VAR=""` (explicitly blank) and `VAR` unset cannot use this helper and must call `os.Getenv` directly.
- `GetStringFromEnv` does not emit a warning when the env var is absent (unlike `GetIntFromEnv` and `GetBoolFromEnv` for invalid values), so there is no diagnostic signal when an expected variable is missing.
- Adding two new exported functions to `pkg/envutil` slightly grows the stable API surface that future refactors must maintain.

#### Neutral
- The `warn` helper is package-private; callers outside `pkg/envutil` cannot reuse it for their own typed-parsing scenarios.
- Boolean env vars that previously used non-canonical comparisons (e.g., `os.Getenv("CI") != ""`) now use `strconv.ParseBool`, which does *not* treat a non-empty non-boolean string as truthy — a subtle behavioral change that is intentional and documented in the PR.
- All three new and existing helpers share the same `debugLog *logger.Logger` optional parameter convention; passing `nil` suppresses structured logging and falls back to `os.Stderr` warnings.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
