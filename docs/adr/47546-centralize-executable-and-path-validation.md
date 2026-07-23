# ADR-47546: Centralize Executable and Path Validation in pkg/fileutil

**Date**: 2026-07-23
**Status**: Draft
**Deciders**: Unknown (Copilot SWE agent, pelikhan)

---

### Context

Static analysis (Sighthound) flagged multiple CWE-78 (OS Command Injection) risks across the codebase where runtime-derived paths or executable names flowed directly into `exec.Command` calls without consistent validation. The affected call sites — in `pkg/cli/runner_guard.go`, `pkg/cli/upgrade_command.go`, `pkg/workflow/dependabot.go`, `pkg/workflow/pip_validation.go`, and `pkg/gitutil/gitutil.go` — each performed ad-hoc inline validation (e.g., `filepath.IsAbs()` check, bare `exec.LookPath()` without further verification), leading to inconsistent security guarantees and duplicated logic. Additionally, the `HEAD:<path>` git-show construction in `ReadFileFromHEAD` concatenated a runtime-derived relative path directly into a git argument, which static analysis also flagged as an injection risk.

### Decision

We will centralize executable and path validation into two new functions in `pkg/fileutil` — `ValidateExecutablePath` and `ResolveExecutablePath` — and update all affected call sites to use them. `ResolveExecutablePath` resolves a bare executable name via `exec.LookPath`, ensures the result is absolute, resolves symlinks, and confirms the file exists and is executable. For git HEAD reads, we will replace the single `git show HEAD:<path>` call with a two-step approach: first use `git ls-tree` with a `:(literal)` pathspec to resolve the blob object ID, then read it with `git cat-file blob <id>`, eliminating the path string from argument construction.

### Alternatives Considered

#### Alternative 1: Per-site ad-hoc validation (status quo)

Each call site continues to perform its own inline checks (e.g., `filepath.IsAbs`, bare `exec.LookPath`). This was the existing approach. It was rejected because checks were inconsistent across sites (some missing symlink resolution, executable-bit checks, or directory rejection), and each new call site is likely to repeat the same partial validation pattern.

#### Alternative 2: Suppress scanner findings with `#nosec` annotations

Apply `#nosec G204` suppression comments at each flagged site without changing validation logic. This was rejected because it would silence the scanner without reducing the actual attack surface; the inconsistent validation would remain.

#### Alternative 3: Third-party security library for path validation

Use an external library (e.g., a path-sanitization or sandboxing package) for validation. This was not chosen because the validation logic required is straightforward and Go's standard library (`filepath.EvalSymlinks`, `os.Stat`, `os/exec.LookPath`) provides all the necessary primitives, avoiding an external dependency for a small, well-understood concern.

### Consequences

#### Positive
- Validation logic is defined once and tested in isolation (`pkg/fileutil/executable_test.go`), making it easier to audit and strengthen consistently.
- Scanner-visible CWE-78 surface is reduced at all affected call sites by enforcing absolute paths, symlink resolution, and executable-bit checks before any `exec.Command` invocation.
- The two-step blob lookup for git HEAD reads eliminates path string concatenation in git arguments entirely, removing a separate class of injection risk.
- `filepath.IsLocal` replaces the fragile `relDir != ".."` prefix check in runner-guard, using the standard library's authoritative locality predicate.

#### Negative
- `ValidateExecutablePath` performs a filesystem stat at call time, which means the executable must exist on disk when the function is called; this can complicate test setups that do not have real executables available (currently handled with `t.TempDir` fixtures, but future callers must be aware).
- Multiple packages now import `pkg/fileutil` (`pkg/cli`, `pkg/gitutil`, `pkg/workflow`), increasing coupling to this utility package; future breaking changes to its API will require coordinated updates across callers.
- The two-step git blob lookup adds a `git ls-tree` call for each `ReadFileFromHEAD` invocation, increasing subprocess overhead (minor, as this path is not on the hot path).

#### Neutral
- All affected `#nosec G204` suppression comments were updated to reflect the new validation path, keeping suppression rationale accurate.
- No user-facing behavior changes: the code keeps existing behavior while tightening internal validation.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
