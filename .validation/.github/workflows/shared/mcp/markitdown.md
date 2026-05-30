---
mcp-servers:
  markitdown:
    container: "mcp/markitdown"
    # Security decision (2026-05-19): keep wildcard while markitdown tool names are versioned upstream.
    # This server is used for read-only content conversion in summarization workflows.
    allowed: ["*"]
---
