---
"gh-aw": patch
---

Update the default Copilot CLI to 1.0.98, including the compatible CLI cache ceiling and installer fallback. This release includes fixes for MCP tool selection, unattended hook permissions, resumed tool calls, and command hook completion.

Retain model-dispatch completion events in SDK session artifacts so unified-session normalization does not lose their native evidence. Defer live shell-output capture until redaction protects credentials split across chunks, and ignore callbacks after the event stream closes. Preserve partial file-edit details and human-response provenance without treating them as session completion or authorization.

Clarify that Copilot SDK mode is stable and non-experimental, and use the artifact-backed temporary directory in its smoke workflow.
