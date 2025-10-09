---
"gh-aw": patch
---

Update rendering of tools in summary tag to use HTML code elements

The Claude and Copilot JavaScript log parsers now use HTML `<code>` elements instead of markdown backticks when rendering tool calls in HTML `<summary>` tags. This ensures proper code styling with monospace font, light gray background, and rounded borders, since summary content is not rendered as markdown.
