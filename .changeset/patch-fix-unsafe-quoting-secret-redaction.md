---
"gh-aw": patch
---

Security fix: Prevent injection vulnerability in secret redaction YAML generation

Fixed a critical security vulnerability (CodeQL go/unsafe-quoting) where secret names containing single quotes could break out of enclosing quotes in generated YAML strings, potentially leading to command injection, SQL injection, or code injection attacks. Added proper escaping via a new `escapeSingleQuote()` helper function that sanitizes secret references before embedding them in YAML.
