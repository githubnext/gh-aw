---
mcp-servers:
  microsoftdocs:
    url: "https://learn.microsoft.com/api/mcp"
    # Security decision (2026-05-19): keep wildcard for this read-only public documentation server.
    # This aligns with least-privilege intent while preserving broad docs lookup capability.
    allowed: ["*"]
---
