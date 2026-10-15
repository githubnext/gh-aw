# ADR-69112: Detect a missing Docker executable before running optional checks

**Date**: 2026-02-11
**Status**: Draft
**Deciders**: Unknown (drafted from PR #69112 by the Design Decision Gate)

---

### Context

Optional Docker-backed static analysis (zizmor, poutine, actionlint, runner-guard, syft, grype, grant, yamllint) is prepared through `pkg/cli/docker_images.go`. When Docker was not installed at all, every helper still shelled out to `docker`, so each call produced its own `exec` failure and `CheckAndPrepareDockerImages` could still kick off background image pulls that were guaranteed to fail and be retried. In long-lived processes — notably the MCP `compile` tool, which may handle many requests — that produced repeated verbose diagnostics for a condition that is constant for the lifetime of the process. The unavailability message was also built by enumerating every requested tool with a `tool: false` suggestion, which made a skipped optional check read like a hard configuration error. Required checks (`validate_images: true`) must keep failing when Docker is unavailable, and informational output must never pollute JSON stdout.

### Decision

We will probe for the `docker` executable on `PATH` with `exec.LookPath` before any Docker invocation, via a new `isDockerCLIAvailable()` helper, and treat "Docker not installed" as a distinct, cheap, early-exit condition. The helper prints a single informational notice to stderr per process, guarded by a package-level `sync.Once` (`dockerMissingNotice`), and `StartDockerImageDownload` returns `started == false` with a `DockerUnavailableError` join function instead of launching a retrying pull. The verbose per-tool error construction in `CheckAndPrepareDockerImages` is replaced by one constant, `dockerUnavailableMessage`, which is also the message surfaced in the MCP `docker_unavailable` warning, so callers see one concise and stable string.

### Alternatives Considered

#### Alternative 1: Keep invoking `docker info` and deduplicate the error output

We could have left the existing call path intact and simply suppressed repeated error text (for example by caching the first failure). This was a close call because it is a smaller diff, but it still pays the cost of a failed `exec` on every check and, more importantly, it does not prevent `StartDockerImageDownload` from spawning doomed background pulls with their own retry loops. The `LookPath` probe distinguishes "not installed" from "installed but daemon down" cheaply and at the correct layer.

#### Alternative 2: Cache the Docker availability result in `pullState`

Storing a resolved availability flag in the existing `pullState` struct would also collapse repeated probes. It was rejected because availability can legitimately change during a process (the new `TestPrepareDockerImagesMissingBeforeDownload` test deliberately removes the binary mid-run), and a sticky cache would make the mock-based test helpers and `ResetDockerPullState` semantics harder to reason about. `sync.Once` is scoped to the user-facing notice only, not to the availability decision itself.

### Consequences

#### Positive

- A machine without Docker gets exactly one informational line on stderr instead of a stream of errors, and stdout stays clean for JSON consumers.
- No futile image pulls or retries are started when the Docker CLI is absent, saving time and avoiding misleading "being downloaded" states.
- The MCP `compile` tool returns a stable `docker_unavailable` warning message, so repeated requests produce byte-identical results (asserted by the new repeated-call test).

#### Negative

- The actionable remediation hints are gone: the old error named each requested tool and suggested `zizmor: false`-style parameters, while the new constant message does not, so users must consult documentation to learn how to disable a check.
- `sync.Once` is process-global state, which requires an explicit reset hook in test helpers (`dockerMissingNotice = sync.Once{}`) and means the notice is emitted only once even across logically unrelated operations in a long-lived process.
- A `PATH` lookup now precedes each availability check, adding a small amount of filesystem work on the hot path.

#### Neutral

- Mock-driven code paths (`mockAvailableInUse`, `SetMockDockerAvailable`) short-circuit before the `LookPath` probe, so existing mocked tests are unaffected; one test now relies on a real empty `PATH` instead of the mock.
- Required image validation (`validate_images: true`) keeps its fail-closed behaviour; only optional/skippable checks change in verbosity.
- New tests depend on a POSIX shell for fake `docker` binaries and are skipped on Windows.
- Behaviour is documented in `docs/src/content/docs/reference/compilation-process.md`.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
