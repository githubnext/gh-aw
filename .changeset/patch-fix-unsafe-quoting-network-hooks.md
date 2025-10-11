---
"gh-aw": patch
---

Fixed unsafe quoting vulnerability in network hook generation (CodeQL Alert #9)

Implemented proper quote escaping using `strconv.Quote()` when embedding JSON-encoded domain data into Python script templates. This prevents potential code injection vulnerabilities (CWE-78, CWE-89, CWE-94) that could occur if domain data contained special characters. The fix uses Go's standard library for safe string escaping and adds `json.loads()` parsing in the generated Python scripts for defense in depth.
