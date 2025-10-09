---
"gh-aw": patch
---

Update Copilot log parser to render tool calls with 6 backticks and structured format

Updated the Copilot log parser to render tool calls using HTML details elements with 6 backticks and structured Parameters/Response sections. Tool responses are now displayed in full (no 500-character truncation) with clear separation between parameters and responses, matching the Codex parser implementation for consistency and better debugging experience.
