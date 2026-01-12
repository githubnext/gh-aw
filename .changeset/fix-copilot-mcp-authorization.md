---
"gh-aw": patch
---

Fix Copilot MCP authorization: add Bearer prefix to Authorization header. The GitHub Copilot CLI expects the "Bearer" authentication scheme in the Authorization header per standard HTTP authentication patterns. The MCP Gateway outputs Authorization headers without the "Bearer" prefix per MCP Gateway Specification v1.3.0 Section 7.1, so the converter script now adds the prefix to ensure compatibility with Copilot CLI.
