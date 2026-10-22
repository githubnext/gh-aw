---
"gh-aw": patch
---

Track agent-origin provenance in the issue-body AST and add optional coarse redaction policies for titles, body content, and agent-authored footer diagnostics while preserving framework attribution and metadata without relaxing existing sanitization.

Require a verified redaction pass before rendering report trees and fail closed if secret-tainted strings remain in the tree or concrete output.

Automatically replace oversized agent-origin subtrees to fit publication limits while preserving complete markers, footers, and diagnostics. Treat structured JSON as an atomic AST node, retain bounded input and mention/link limits, and fail before publishing if protected content alone cannot fit.
