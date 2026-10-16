---
"gh-aw": patch
---

Update the Pydantic AI smoke and conformance imports to the upstream definition for harness and CLAI 2 version 0.56.0. This replaces the legacy `pai` runner with headless CLAI 2, preserves model routing and gateway MCP tools, and records canonical session events including native tool results.

Decode the upstream keyed canonical frames in the Pydantic AI session adapter so local audits and unified sessions retain observed tool arguments, outcomes, reasoning, identity, and usage. Legacy stdout capture remains explicitly partial.
