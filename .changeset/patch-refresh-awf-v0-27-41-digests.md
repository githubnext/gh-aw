---
"gh-aw": patch
---

Refresh container image digest pins for `ghcr.io/github/gh-aw-firewall` v0.27.41. The images were rebuilt to upgrade the Go runtime (addressing Go stdlib CVEs) and update the Ubuntu 22.04 base layer. Updated digests for agent, api-proxy, cli-proxy, and squid; also added the missing agent-act pin for this version.
