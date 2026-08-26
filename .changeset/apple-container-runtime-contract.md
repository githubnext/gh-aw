---
"gh-aw": minor
---

Add the compiler/runtime contract for the `apple-container` sandbox runtime (AWF's Apple Virtualization.framework workload preview, gh-aw-firewall#7764).

`sandbox.agent.runtime: apple-container` is now a recognised runtime profile: network-isolated, rootless AWF invocation, no host access, and no compiler-generated runtime installation. When selected, the compiler emits **both** required AWF selectors — `container.containerRuntime: "apple-container"` and `appleContainer.previewEnabled: true` — and suppresses `network.topologyAttach`, which AWF rejects for this runtime because externally owned peers are not published to macOS loopback. Docker is still required: AWF keeps Squid, the API proxy, the CLI proxy, and the MCP gateway in Docker Compose on the host, and only the agent workload moves to Apple Container.

The runtime is gated behind a new `AWFAppleContainerMinVersion` (v0.28.9) that is deliberately above the default AWF version, so a workflow must explicitly pin `sandbox.agent.version` (or `firewall.version`) to an AWF build that understands the runtime. The selector and the `appleContainer` section are emitted as a unit and never reach an older AWF.

Compile-time validation fails closed on everything gh-aw can know:

- **Runner.** `apple-container` requires a self-hosted bare-metal Apple Silicon host (Darwin arm64, macOS 26+, `kern.hv_support=1`), declared explicitly as `runs-on: [self-hosted, macOS, ARM64]` (extra pool labels allowed). GitHub-hosted `macos-*` labels — including the `-xlarge` Apple Silicon images — are rejected because they are virtual machines without nested virtualization. Runner groups, expressions, conflicting OS/arch labels, and an omitted `runs-on` are rejected rather than guessed. The blanket macOS rejection stays in place for every other runner field and for every other runtime.
- **Features.** `runner.topology: arc-dind`, enclaves, `sandbox.agent.mounts`, `filesystem.allowWrite`, `ssl_bump`, Vertex AI credential isolation, `allow-host-ports`, GitHub Actions `services:` port mappings, `runtime-install`, and the raw AWF arguments AWF refuses under this runtime (`--legacy-security`, `--enable-host-access`, `--dns-over-https`, `--topology-attach`, `--dind`, `--build-local`, `--agent-image`, `--sysroot-image`, `--volume`, `--tty`, `--ssl-bump`, and others) all produce actionable errors.

A new `appleInit` role is added to the `sandbox.agent.images` manifest and to `container.images`, and is required whenever the runtime is selected: Apple Container maintains a separate image store, so it is excluded from Docker predownload.

This layer intentionally does not generate runnable Apple Container setup — runtime installation, image provisioning, and MCP transport rewiring follow separately.
