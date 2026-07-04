---
"gh-aw": patch
---

Fix firewall image digest pinning regression for default AWF version (v0.27.22): ensure all four `gh-aw-firewall` sidecar images (`agent`, `api-proxy`, `cli-proxy`, `squid`) are digest-pinned in compiled lock files via the embedded container pins table. Adds regression tests covering the default version including the new `cli-proxy` image introduced in v0.82.
