---
"githubnext/gh-aw": patch
---

Remove Agentic Run Information from step summary

The "Agentic Run Information" section is no longer displayed in the GitHub Actions step summary. The information remains available in the action logs via console.log() and as an artifact (aw_info.json), reducing noise in the step summary while preserving accessibility for workflows that need this metadata.
