# ADR-44796: Use gVisor (runsc) as an Optional Container Runtime for Agent Containers

**Date**: 2026-07-10
**Status**: Draft
**Deciders**: @lpcox

---

### Context

Agent containers in gh-aw run under a standard Docker runtime backed by Linux cgroups and namespaces. This provides process-level isolation but leaves the host kernel's system call surface fully accessible to agent code. Any vulnerability in a system call handler can be exploited to escape the container. For higher-assurance workloads, operators need a runtime that interposes on all syscalls and runs them through a user-space kernel, substantially reducing the kernel attack surface exposed to untrusted agent code. The AWF daemon already supports `container.containerRuntime: "gvisor"` in its stdin config (gh-aw-firewall#6093), but the compiler had no way to emit that setting or install the runtime on the runner. This PR wires up the missing compiler side: a new `sandbox.agent.runtime` frontmatter field, an auto-generated pre-agent gVisor install step, and the AWF config JSON plumbing that passes `containerRuntime: "gvisor"` to AWF.

### Decision

We will add `sandbox.agent.runtime: gvisor` as an optional frontmatter field. When set, the compiler (1) emits a GitHub Actions step before the AWF install step that downloads `runsc` and `containerd-shim-runsc-v1` from `storage.googleapis.com/gvisor`, installs them to `/usr/local/bin`, registers them with Docker via `sudo runsc install`, restarts Docker with `systemctl restart`, and verifies the runtime with a test container, and (2) passes `containerRuntime: "gvisor"` in the AWF stdin config JSON so AWF starts the agent container under `runsc` — once the effective AWF version is at or above `AWFContainerRuntimeMinVersion` (`v0.27.30`, released in gh-aw-firewall#6093). One incompatible combination is rejected at compile time: `runtime: gvisor` + `runner.topology: arc-dind` (systemd unavailable on ARC DinD). The `runtime: gvisor` option is compatible with both `sudo: false` (default, network-isolation mode) and `sudo: true` — the gVisor install step uses shell-level sudo commands independently of the `sandbox.agent.sudo` setting. The existing `sudo: true` deprecation warning continues to apply regardless of `runtime`.

### Alternatives Considered

#### Alternative 1: Standard Docker runtime (status quo)

Continue using only Linux cgroups/namespaces for agent container isolation without adding a new runtime option. System calls from inside the container reach the host kernel directly. This approach requires zero compiler changes and no runner setup time, and it already satisfies many workloads. It was rejected because it does not meet the kernel-level isolation requirement: a single exploited syscall can escape the container, and operators running sensitive workloads need a stronger isolation boundary.

#### Alternative 2: Kata Containers

Kata Containers provide VM-level isolation — each container runs in a lightweight QEMU/Firecracker VM with a dedicated kernel. This gives stronger isolation guarantees than gVisor in some threat models. It was not chosen because: (a) Kata requires KVM hardware virtualization support, which is not available in all GitHub-hosted runner configurations; (b) the AWF daemon already has native support for `containerRuntime: "gvisor"` but not for Kata; and (c) gVisor's installation footprint (two binaries + `runsc install`) is far smaller than Kata's, reducing runner setup time and dependency surface.

#### Alternative 3: Seccomp / AppArmor profiles

Reduce the syscall surface by attaching a restrictive seccomp profile or AppArmor policy to the agent container, without changing the OCI runtime. This has near-zero overhead and no special hardware requirements. It was not chosen because it still routes all permitted syscalls through the real host kernel — a zero-day in a whitelisted syscall path remains exploitable. Profiles also require ongoing maintenance as agent behavior evolves, and they do not satisfy an isolation requirement that explicitly demands a user-space kernel intercept layer.

### Consequences

#### Positive
- Agent containers configured with `runtime: gvisor` run under gVisor's `runsc` OCI runtime, which interposes on all system calls via a user-space kernel, substantially reducing the host kernel attack surface.
- Compile-time validation catches the known incompatible combination (`arc-dind`) before a workflow is submitted, giving authors a clear error message rather than a runtime failure.
- The change is fully backward-compatible: existing workflows that do not set `sandbox.agent.runtime` are unaffected and continue to run under the default Docker runtime.
- `runtime: gvisor` is compatible with `sudo: false` (the default, network-isolation mode): the install step uses shell-level sudo commands independently of the `sandbox.agent.sudo` setting, so operators can have both gVisor kernel isolation and AWF network isolation.
- Removing the special case for `AgentRuntimeGVisor` in `validateStrictSandboxCustomization` simplifies the strict-mode validation path.

#### Negative
- `runtime: gvisor` is incompatible with `runner.topology: arc-dind`, restricting which runner configurations can use this feature.
- The gVisor install step adds observable setup time (binary downloads, Docker restart, verification container run) to every workflow execution that enables this option.
- The `containerRuntime` field in the AWF config is gated behind `AWFContainerRuntimeMinVersion` (`v0.27.30`, shipped in gh-aw-firewall#6093); workflows pinning an older version will not emit this field.

#### Neutral
- The `systemctl restart docker` (rather than `reload`) constraint documented in the implementation will need to be preserved in future modifications to the install step: Docker's SIGHUP reload does not call `setHostGatewayIP()`, which breaks `--add-host host.docker.internal:host-gateway`.
- AWF translates the `"gvisor"` string in its config to `"runsc"` internally; the compiler intentionally passes the human-readable label rather than the internal runtime name.
- Future container runtime options (e.g., Kata) can follow the same `AgentRuntime` type + `sandbox.agent.runtime` enum pattern established here.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
