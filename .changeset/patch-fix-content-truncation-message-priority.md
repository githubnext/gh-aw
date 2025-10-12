---
"gh-aw": patch
---

Fix content truncation message priority in sanitizeContent function

Fixed a bug where the `sanitizeContent` function was applying truncation checks in the wrong order. When content exceeded both line count and byte length limits, the function would incorrectly report "Content truncated due to length" instead of the more specific "Content truncated due to line count" message. The truncation logic now prioritizes line count truncation, ensuring users get the most accurate truncation message based on which limit was hit first.
