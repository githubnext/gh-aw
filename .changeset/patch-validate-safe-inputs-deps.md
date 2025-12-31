---
"gh-aw": patch
---

Fail-fast validation for safe-inputs MCP server startup and stricter error handling in setup scripts.

Validates that all required JavaScript dependency files for the safe-inputs MCP server are present before starting the server, lists missing files and directory contents when validation fails, and changes setup scripts to treat missing files as errors and exit immediately.

This prevents the server from starting with missing dependencies and producing opaque Node.js MODULE_NOT_FOUND crashes.

