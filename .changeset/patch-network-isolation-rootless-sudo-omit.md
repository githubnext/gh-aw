---
"gh-aw": patch
---

Omit `sudo` from generated lock.yml when `sandbox.agent.network-isolation: true` is set.

In network-isolation mode, AWF uses a container-based egress topology that does not
require `NET_ADMIN` or host-iptables enforcement, so the binary can run rootless —
a key requirement for ARC (Actions Runner Controller) Kubernetes runners where
passwordless `sudo` is often unavailable.

Changes:
- `install_awf_binary.sh` gains a `--rootless` flag that substitutes every `sudo`
  call with a direct invocation, allowing the AWF binary to be installed without root.
- The compiler passes `--rootless` to the install script when network-isolation is enabled.
- `GetAWFCommandPrefix` now emits `awf` (no `sudo -E`) when network-isolation is active.
- The `sudo chmod -R a+rX` permission-fix in the "Print firewall logs" step is skipped
  for network-isolation workflows, since rootless AWF does not create root-owned files.
- Legacy (non-isolation) topology is unchanged — it still uses `sudo` as before.

Also bumps the default pinned container versions to the latest releases:
- Firewall (`ghcr.io/github/gh-aw-firewall/{agent,api-proxy,squid}`): `v0.27.10`
- MCP gateway (`ghcr.io/github/gh-aw-mcpg`): `v0.3.30`
