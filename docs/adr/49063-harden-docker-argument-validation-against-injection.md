# ADR-49063: Harden Docker Argument Validation Against Command Injection

**Date**: 2026-07-30
**Status**: Draft
**Deciders**: Unknown (security-triggered fix by copilot-swe-agent, initiated by Sighthound scan)

---

### Context

Sighthound's static security scan flagged 33 Critical CWE-78 (OS Command Injection) findings in the `pkg/cli` package. Five high-severity findings were in the Docker exec paths of `runner_guard.go`, `grant.go`, `poutine.go`, and the self-relaunch path of `upgrade_command.go`. In these paths, Docker volume mount strings were assembled by direct string concatenation (e.g., `gitRoot + ":/workdir"`), and exec arguments were passed without sanitization checks for control characters or NUL bytes. While `exec.Command` avoids shell interpretation, unsanitized arguments can still inject unexpected Docker flags or cause undefined behavior in downstream processes. The fix needed to be applied consistently across multiple scanners with minimal duplication.

### Decision

We will centralize Docker `-v` argument construction into shared helpers (`buildDockerVolumeMount`, `buildDockerReadonlyFileMount`) in `pkg/cli/docker_args_validation.go` that enforce absolute-path validation, container-path normalization, control-character rejection, and file-type checks. All Docker exec sites in `grant.go`, `poutine.go`, and `runner_guard.go` will use these helpers instead of string concatenation. The self-relaunch path in `upgrade_command.go` will reject any argument containing a NUL byte before passing it to `exec.Command`. The `grant.go` docker executable will be resolved via `fileutil.ResolveExecutablePath` to prevent PATH-hijacking.

### Alternatives Considered

#### Alternative 1: Per-function inline validation (status quo + extensions)

Each scanner function continues to perform its own validation inline, as `runner_guard.go` already did for absolute-path checks. Validation logic would be duplicated across `grant.go`, `poutine.go`, and `runner_guard.go`. Not chosen because it produces inconsistent coverage (each site may miss different edge cases), makes auditing harder, and does not consolidate error handling in a single testable unit.

#### Alternative 2: Shell-escape / shlex sanitization before concatenation

Apply a shlex-style escaping function to volume mount components before string concatenation, similar to how some container runtimes sanitize arguments. Not chosen because `exec.Command` already avoids shell interpretation — the risk is not shell metacharacters but rather structural injection into Docker's `-v` argument parsing and NUL bytes in argument vectors. Escaping addresses the wrong threat model and masks invalid input rather than rejecting it.

### Consequences

#### Positive
- Eliminates the CWE-78 Critical findings in Docker exec paths by ensuring all volume mount strings are constructed from validated, normalized, absolute paths.
- Centralizes validation logic into a single package-internal file, making future audits and policy changes a one-location edit.
- Adds explicit NUL byte rejection in the self-relaunch path, preventing argument injection in process re-exec scenarios.
- Adds unit tests for the shared helpers and for the image reference / NUL byte validations, increasing coverage of security-critical paths.

#### Negative
- Each `buildDockerReadonlyFileMount` call now performs an `os.Stat` syscall to verify the host file is a regular file; this adds a small overhead on every Docker scanner invocation.
- The new helpers return errors that all callers must propagate, increasing the error-path surface in functions that previously could not fail at mount-construction time.
- Callers must now satisfy stricter pre-conditions (normalized absolute paths); any future refactor that produces relative paths will fail at runtime rather than silently passing an invalid mount string.

#### Neutral
- The `pkg/cli` package gains a new internal dependency on `pkg/fileutil` (already a transitive dependency via other files in the package).
- `grant.go` now resolves the `docker` binary via `fileutil.ResolveExecutablePath` rather than relying on the ambient PATH; this aligns it with how `runner_guard.go` already worked, but is a behavioral change that could surface errors on systems where `docker` is not on PATH.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
