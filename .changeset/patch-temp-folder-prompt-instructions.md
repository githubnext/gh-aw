---
"gh-aw": patch
---

Add temporary folder usage instructions to agentic workflow prompts

Agentic workflows now include explicit instructions for AI agents to use `/tmp/gh-aw/agent/` for temporary files instead of the root `/tmp/` directory. This improves file organization and prevents conflicts between workflow runs.
