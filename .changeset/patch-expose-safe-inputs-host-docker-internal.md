---
"gh-aw": patch
---

Expose the safe-inputs MCP HTTP server via Docker's `host.docker.internal` and add `host.docker.internal` to the Copilot firewall allowlist so containerized services (like the AWF firewall) can access host-hosted safe-inputs.
