---
"gh-aw": patch
---

Compute reusable-workflow artifact prefixes and resolve host repository metadata within each consuming job so secret masking cannot blank their cross-job transport. Missing host repository metadata fails explicitly instead of redirecting cross-repository operations to the caller.

Avoid duplicate metadata steps for reusable upload-asset jobs, scope dedicated uploader App tokens to the resolved repository, and reject SARIF or coverage targets that the pinned upload actions cannot honor instead of publishing to the caller repository or revision.
