---
"gh-aw": minor
---

Add an extensible Actions/AW AST lint-rule framework and reusable source/sink/filter taint analysis. Run the first built-in checker with `gh aw lint --rules secret-leak`, list rules with `--list-rules`, and export value-free diagnostics with `--json`. Apply the rules to repository workflows in regression tests and move the Gemini inventory API key from its request URL to an authentication header.
