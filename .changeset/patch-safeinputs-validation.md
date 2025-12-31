---
"gh-aw": patch
---

Add validation for safe-inputs MCP server dependencies

Improves reliability of safe-inputs MCP server startup by adding comprehensive dependency validation:
- Validates all 12 required dependency files exist before starting the server
- Fails fast with clear error messages if files are missing
- Changes warnings to errors in setup script to prevent silent failures
- Adds proper shell script error handling (shebang, set -e, cd || exit patterns)

This prevents cryptic Node.js module errors when dependencies are missing and makes debugging easier.
