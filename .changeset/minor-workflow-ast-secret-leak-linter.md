---
"gh-aw": minor
---

Add an extensible Actions/AW AST lint-rule framework and reusable source/sink/filter taint analysis. Run all registered AST checks alongside actionlint by default with `gh aw lint`, select specific checks with `--rules secret-leak`, list rules with `--list-rules`, and export value-free diagnostics with `--rules secret-leak --json`. Apply the rules to repository workflows in regression tests and move the Gemini inventory API key from its request URL to an authentication header.

Add threat-spec-mapped AST rules for action pinning, direct template injection, workflow-run scope, and literal job timeouts. Preserve independent diagnostics when another rule fails, and document the boundary between AST evidence and full compiler/runtime conformance.
