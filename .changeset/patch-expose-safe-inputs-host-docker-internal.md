---
"gh-aw": patch
---

Expose the safe-inputs MCP HTTP server via Docker's `host.docker.internal` and
include `host.docker.internal` in the Copilot firewall allowlist when
safe-inputs is enabled. This fixes access from the AWF firewall container to
the host-hosted safe-inputs service.

Changes include using `host.docker.internal` for rendered safe-inputs URLs and
updating domain calculation logic to add the domain when safe-inputs is
enabled.

