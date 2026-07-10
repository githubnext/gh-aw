---
"gh-aw": minor
---

Add `sandbox.agent.runtime: gvisor` frontmatter field for gVisor container runtime support.

When set to `gvisor`, the compiler:
1. Emits a pre-agent gVisor installation step that downloads, installs, and verifies `runsc` and `containerd-shim-runsc-v1`
2. Passes `container.containerRuntime: "gvisor"` in the AWF stdin config JSON so AWF starts the agent container under the `runsc` runtime

Compile-time validation rejects incompatible combinations:
- `sandbox.agent.runtime: gvisor` + `runner.topology: arc-dind` (gVisor requires `systemctl restart docker` which is unavailable on ARC DinD runners)
- `sandbox.agent.runtime: gvisor` + `sandbox.agent.sudo: false` (the install step requires root access)

When `runtime: gvisor` is configured, `sandbox.agent.sudo: true` is required and the `sudo: true` deprecation warning/error is suppressed since gVisor fundamentally requires root.
