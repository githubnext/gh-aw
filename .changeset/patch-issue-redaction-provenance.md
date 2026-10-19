---
"gh-aw": patch
---

Track agent-origin provenance in the issue-body AST and add optional coarse redaction policies for titles, body content, and agent-authored footer diagnostics while preserving framework attribution and metadata without relaxing existing sanitization.

Require a verified redaction pass before rendering report trees and fail closed if secret-tainted strings remain in the tree or concrete output.
