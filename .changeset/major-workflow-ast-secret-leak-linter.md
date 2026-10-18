---
"gh-aw": major
---

Add an extensible Actions/AW AST lint-rule framework and reusable source/sink/filter taint analysis. Run all registered AST checks alongside actionlint by default with `gh aw lint`, select specific checks with `--rules secret-leak`, list rules with `--list-rules`, and export value-free diagnostics with `--rules secret-leak --json`. Apply the rules to repository workflows in regression tests and move the Gemini inventory API key from its request URL to an authentication header.

Add threat-spec-mapped AST rules for action pinning, direct template injection, workflow-run scope, and literal job timeouts. Preserve independent diagnostics when another rule fails, and document the boundary between AST evidence and full compiler/runtime conformance.

Add permission-scope checks for blanket write grants and invalid OIDC reads, plus companion-AW source rules for disabled version checks, conditional imports, and private-to-public flow opt-outs. Respect authored strict policy without inferring job authorization or missing source configuration.

Model value-free shell predicates, no-op builtins, and presence-only alternate expansions without spurious secret-flow warnings. Scope file-coverage warnings to possible modeled secret flows rather than unrelated public file operations. Preserve disclosures through tracing, nested commands, and parameter-expansion errors; include script-relative locations for unmodeled command warnings.

**Breaking change:** `gh aw lint` now runs built-in AST rules by default, so workflows that previously passed actionlint-only linting may report AST findings and return a nonzero exit status. Review and fix those findings, or run `actionlint` directly on the generated `.lock.yml` files to retain actionlint-only behavior. `gh aw lint --rules=...` selects AST rules and skips actionlint; it is not an actionlint-only mode.
