---
"gh-aw": patch
---

Expose the safe-inputs MCP HTTP server via Docker's `host.docker.internal` and add `host.docker.internal` to the Copilot firewall allowlist when safe-inputs is enabled.

This enables the AWF firewall container to access host-hosted safe-inputs services.

