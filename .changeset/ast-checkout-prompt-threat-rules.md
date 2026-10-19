---
"gh-aw": patch
---

Add two authored-AW AST threat linters: `pull-request-target-checkout` (CTR-008) checks trusted checkout refs, exact allowed repository/ref pairs, and additional fetches; `prompt-expression` (CTR-010) rejects multiline or malformed prompt expressions and blocked JavaScript property traversal. Share checkout predicates with the compiler and expression parsing with secret-flow analysis. Preserve import uncertainty, strict/non-strict checkout severity, and value-free diagnostics without claiming full compiler or runtime conformance.
