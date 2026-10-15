---
"gh-aw": patch
---

Compute reusable-workflow artifact prefixes and resolve host repository metadata within each consuming job so secret masking cannot blank their cross-job transport. Missing host repository metadata fails explicitly instead of redirecting cross-repository operations to the caller.
