---
"gh-aw": patch
---

Expose the safe-inputs MCP HTTP server via Docker's `host.docker.internal` and add `host.docker.internal` to the Copilot firewall allowlist so containerized services can access host-hosted safe-inputs.

This is a patch-level change (internal/tooling) and does not introduce breaking changes.

