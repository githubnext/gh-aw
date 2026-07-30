# ADR-49063: Harden Docker Argument Validation Against Command Injection

**Date**: 2026-07-30
**Status**: Accepted
**Deciders**: pelikhan, Copilot

---

### Context

Sighthound's static security scan flagged 33 Critical CWE-78 (OS Command Injection) findings in the `pkg/cli` package. Five high-severity findings were in the Docker exec paths of `runner_guard.go`, `grant.go`, `poutine.go`, and the self-relaunch path of `upgrade_command.go`. In these paths, Docker volume mount strings were assembled by direct string concatenation (for example, `gitRoot + ":/workdir"`), and some callers still relied on `exec.Command("docker", ...)` PATH resolution at launch time. While `exec.Command` avoids shell interpretation, unsanitized mount arguments can still be re-parsed by Docker's colon-delimited `-v` syntax, and copyable verbose commands can become unsafe when user-controlled values are interpolated without shell quoting. The fix needed to be applied consistently across multiple scanners with minimal duplication.

### Decision

We will centralize Docker `-v` argument construction into shared helpers (`buildDockerVolumeMount`, `buildDockerReadonlyFileMount`) in `pkg/cli/docker_args_validation.go` that enforce absolute-path validation, container-path normalization, rejection of control characters and colon-delimiter injection, and file-type checks. On the host side, callers will reject mount paths containing unsupported `:` characters while preserving Windows drive-letter paths; on the container side, callers will reject any colon because Docker would reinterpret it as an extra mount segment. All Docker exec sites in `grant.go`, `poutine.go`, and `runner_guard.go` will use these helpers instead of string concatenation, and every scanner will resolve the Docker executable through `fileutil.ResolveExecutablePath("docker")` before launch. The self-relaunch path in `upgrade_command.go` will reject any argument containing a NUL byte before passing it to `exec.Command`. Verbose "run this directly" hints for the Grant and Poutine scanners will be rendered with shell-escaped arguments so they remain copyable without reintroducing injection risk in documentation output.

### Alternatives Considered

#### Alternative 1: Per-function inline validation (status quo + extensions)

Each scanner function continues to perform its own validation inline, as `runner_guard.go` already did for absolute-path checks. Validation logic would be duplicated across `grant.go`, `poutine.go`, and `runner_guard.go`. Not chosen because it produces inconsistent coverage (each site may miss different edge cases), makes auditing harder, and does not consolidate error handling in a single testable unit.

#### Alternative 2: Switch all scanners to Docker `--mount` syntax

Replace every `-v host:container[:ro]` mount with `--mount type=bind,src=...,dst=...[,readonly]` so colons in host paths do not need special-case handling. This was rejected for this fix because the existing scanners already share `-v`-style call sites and test expectations, and the smallest safe patch was to harden those exact paths by rejecting ambiguous colon-delimited inputs. A broader `--mount` migration remains possible later if we want to permit more host-path shapes.

#### Alternative 3: Shell-escape / shlex sanitization before concatenation

Apply a shlex-style escaping function to volume mount components before string concatenation, similar to how some container runtimes sanitize arguments. Not chosen because `exec.Command` already avoids shell interpretation — the risk is not shell metacharacters but rather structural injection into Docker's `-v` argument parsing and NUL bytes in argument vectors. Escaping addresses the wrong threat model and masks invalid input rather than rejecting it.

### Consequences

#### Positive
- Eliminates the CWE-78 Critical findings in Docker exec paths by ensuring all volume mount strings are constructed from validated, normalized, absolute paths.
- Centralizes validation logic into a single package-internal file, making future audits and policy changes a one-location edit.
- Prevents ambiguous Docker `-v` parsing by rejecting colon-bearing container paths and unsupported host-path colons before launch, instead of relying on Docker's own mount parser to fail safely.
- Aligns Poutine with Grant and Runner Guard by resolving the Docker executable path explicitly before `exec.Command`, closing the remaining PATH-hijack gap in this hardening pass.
- Adds explicit NUL byte rejection in the self-relaunch path, preventing argument injection in process re-exec scenarios.
- Adds unit tests for the shared helpers, colon/control-character rejection paths, verbose command quoting, and the image reference / NUL byte validations, increasing coverage of security-critical paths.

#### Negative
- Each `buildDockerReadonlyFileMount` call now performs an `os.Stat` syscall to verify the host file is a regular file; this adds a small overhead on every Docker scanner invocation.
- The new helpers return errors that all callers must propagate, increasing the error-path surface in functions that previously could not fail at mount-construction time.
- Callers must now satisfy stricter pre-conditions (normalized absolute paths and unambiguous `-v` mount components); any future refactor that produces relative paths or colon-bearing mount targets will fail at runtime rather than silently passing an invalid mount string.

#### Neutral
- The `pkg/cli` package gains a new internal dependency on `pkg/fileutil` (already a transitive dependency via other files in the package).
- `grant.go` and `poutine.go` now resolve the `docker` binary via `fileutil.ResolveExecutablePath` rather than relying on the ambient PATH; this is a behavioral change that could surface errors on systems where `docker` is not on PATH.

---

*Accepted after implementation and conformance validation.*
