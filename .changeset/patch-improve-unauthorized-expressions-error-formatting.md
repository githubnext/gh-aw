---
"gh-aw": patch
---

Pretty print unauthorized expressions error message with line breaks

When compilation fails due to unauthorized expressions, the error message now displays each expression on its own line with bullet points, making it much easier to read and identify which expressions are valid. Previously, all expressions were displayed in a single long line that was difficult to scan.
