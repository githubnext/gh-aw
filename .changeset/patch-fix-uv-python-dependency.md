---
"gh-aw": patch
---

Fix: Add setup-python dependency for uv tool in workflow compilation

The workflow compiler now correctly adds the required `setup-python` step when the `uv` tool is detected via MCP server configurations. Previously, the runtime detection system would skip all runtime setup when ANY setup action existed in custom steps, causing workflows using `uv` or `uvx` commands to fail.

The fix refactors runtime detection to:
- Always run runtime detection and process all sources
- Automatically inject Python as a dependency when uv is detected
- Selectively filter out only runtimes that already have setup actions, rather than skipping all detection
