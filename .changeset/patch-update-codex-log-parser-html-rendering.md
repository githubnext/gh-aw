---
"gh-aw": patch
---

Update Codex log parser to render tool calls using HTML details with 6 backticks

The Codex log parser now renders tool calls using collapsible HTML `<details>` elements with 6 backticks for code regions, matching the format used in other parsers. This improves debugging by showing tool parameters and responses in an organized, collapsible format with proper syntax highlighting. Tool signatures are now wrapped in `<code>` elements for better visual formatting with monospace font and light gray background.
