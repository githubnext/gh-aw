---
"gh-aw": patch
---

Refactor: Extract duplicate MCP config renderers to shared functions

Eliminated 124 lines of duplicate code by extracting MCP configuration rendering logic into shared functions. The Playwright, safe outputs, and custom MCP configuration renderers are now centralized in `mcp-config.go`, ensuring consistency between Claude and Custom engines while maintaining 100% backward compatibility.
