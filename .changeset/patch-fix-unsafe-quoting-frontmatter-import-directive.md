---
"gh-aw": patch
---

Security Fix: Unsafe Quoting in Import Directive Warning (Alert #8)

Fixed unsafe string quoting in the `processIncludesWithVisited` function that could lead to potential injection vulnerabilities. The fix applies Go's `%q` format specifier to safely escape special characters in deprecation warning messages, replacing the unsafe `'%s'` pattern. This addresses CodeQL alert #8 (go/unsafe-quoting) related to CWE-78 (OS Command Injection), CWE-89 (SQL Injection), and CWE-94 (Code Injection).
