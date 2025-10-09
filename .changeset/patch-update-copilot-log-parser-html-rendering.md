---
"gh-aw": patch
---

Update Copilot log parser to render tool calls using HTML details with 6 backticks

The Copilot log parser now renders tool calls using collapsible HTML `<details>` elements with 6 backticks for code regions, matching the format used in the Codex parser. This improves debugging by showing tool parameters and responses in an organized, collapsible format with proper syntax highlighting. The format now includes separate **Parameters:** and **Response:** sections, with parameters formatted as JSON for better readability.
