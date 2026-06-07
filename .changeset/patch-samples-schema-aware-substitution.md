---
"gh-aw": patch
---

Make compile-time substitution of `${{ ... }}` runtime expressions in `safe-outputs.*.samples` schema-aware: the placeholder is now chosen to satisfy the local schema node (first enum value, `true` for boolean, `1` for number/integer, date stub for `format: date`, etc.), so samples that bind a runtime expression to an enum or non-string field validate correctly.
