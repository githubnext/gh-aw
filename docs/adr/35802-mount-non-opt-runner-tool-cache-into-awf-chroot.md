# ADR-35802: Mount Non-/opt Runner Tool Cache into AWF Chroot

**Date**: 2026-05-29
**Status**: Draft
**Deciders**: Unknown (draft generated from PR #35802)

---

## Part 1 — Narrative (Human-Friendly)

### Context

The AWF firewall sandbox runs the agent engine inside a chroot and only the directories it explicitly bind-mounts are visible to processes inside. The generated command already extends `PATH` by searching both `/opt/hostedtoolcache` and `/home/runner/work/_tool` for `bin` directories, but only `/opt/hostedtoolcache` was actually mounted into the chroot. On runners where `setup-node` installs Node under `RUNNER_TOOL_CACHE` (commonly `/home/runner/work/_tool`) instead of the default `/opt/hostedtoolcache`, the Node binary was on `PATH` in theory but absent from the chroot filesystem, so agent startup failed with `exit 127` ("node runtime missing"). The fix must make non-default tool-cache locations visible inside the sandbox without weakening the firewall's isolation guarantees on the common `/opt` case.

### Decision

We will add a runtime shell probe to the generated AWF command (`BuildAWFCommand` in `pkg/workflow/awf_helpers.go`) that resolves the active tool cache from `RUNNER_TOOL_CACHE` (falling back to `/opt/hostedtoolcache`, then to the legacy `/home/runner/work/_tool` path) and emits a read-only bind-mount (`--mount <path>:<path>:ro`) into the chroot **only when** the resolved cache lives outside `/opt/*`. The mount expression is captured in a `GH_AW_TOOL_CACHE_MOUNT` shell variable and injected into the `sudo -E awf ...` invocation alongside the existing dynamic-args variables, so default `/opt` runners are unaffected while non-standard runners gain Node visibility.

### Alternatives Considered

#### Alternative 1: Unconditionally mount the runner work/_tool directory

We could always add `--mount /home/runner/work/_tool:/home/runner/work/_tool:ro` regardless of the runner. This is simpler with no runtime branching, but it widens the read-only mount surface on every run — including the common `/opt` runners that do not need it — and would not cover arbitrary `RUNNER_TOOL_CACHE` overrides that point elsewhere. Rejected in favor of probing the actual cache location and mounting only when necessary.

#### Alternative 2: Resolve and copy the Node binary into the chroot

Instead of mounting, we could locate the Node binary on the host and copy it (plus its runtime dependencies) into a directory already visible in the chroot. This avoids adding a mount, but copying a dynamically linked runtime and its shared libraries is fragile and version-dependent, and would duplicate logic the existing `PATH`/mount approach already handles. Rejected as more brittle than a read-only bind-mount.

### Consequences

#### Positive
- Agent startup succeeds on runners where the tool cache is outside `/opt` (e.g. `RUNNER_TOOL_CACHE=/home/runner/work/_tool`), eliminating the `exit 127` failure mode.
- The mount is conditional, so default `/opt/hostedtoolcache` runners are unchanged and gain no additional mount surface.
- Honors an explicit `RUNNER_TOOL_CACHE` override rather than hard-coding a single legacy path.

#### Negative
- The generated AWF command grows a multi-line shell probe, increasing its complexity and the maintenance burden of reading/auditing the emitted YAML.
- Correctness depends on runtime shell logic (`[ -d ... ]`, `[[ != /opt/* ]]`) that is validated indirectly via string-contains unit assertions and golden files, not by executing the probe.
- A non-`/opt` cache path is exposed read-only inside the sandbox, slightly widening the chroot's visible filesystem on those runners.

#### Neutral
- All engine golden outputs (`claude`, `codex`, `copilot`, `gemini`, `pi`, and compile fixtures) were regenerated to include the new probe and the `${GH_AW_TOOL_CACHE_MOUNT}` argument.
- The mount variable follows the same injection pattern as the existing `GH_AW_DOCKER_HOST_PATH_PREFIX_ARGS` probe.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Tool Cache Visibility in the AWF Chroot

1. The generated AWF command **MUST** resolve the active tool cache from `RUNNER_TOOL_CACHE`, falling back to `/opt/hostedtoolcache` when it is unset.
2. When the resolved tool cache exists and lies outside `/opt/*`, the generated command **MUST** emit a read-only bind-mount of that path into the chroot.
3. When the resolved tool cache lies under `/opt/*`, the generated command **MUST NOT** emit an additional tool-cache mount.
4. When the resolved tool cache does not exist but the legacy `/home/runner/work/_tool` directory is present, the generated command **SHOULD** emit a read-only bind-mount of `/home/runner/work/_tool`.
5. Tool-cache mounts emitted by this mechanism **MUST** be read-only (`:ro`).
6. The tool-cache mount argument **MUST** be injected into the `awf` invocation via a dedicated shell variable (`GH_AW_TOOL_CACHE_MOUNT`) consistent with the existing dynamic-args injection pattern.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26666085026) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
