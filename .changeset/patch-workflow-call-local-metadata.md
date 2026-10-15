---
"gh-aw": patch
---

Compute reusable-workflow artifact prefixes and resolve host repository metadata within each consuming job so secret masking cannot blank their cross-job transport. Missing host repository metadata fails explicitly instead of redirecting cross-repository operations to the caller.

Avoid duplicate metadata steps for reusable upload-asset jobs, scope dedicated uploader App tokens to the resolved repository, and reject SARIF or coverage targets that the pinned upload actions cannot honor instead of publishing to the caller repository or revision.

Bind partial-rerun downloads to validated same-run producer-attempt descriptors while preserving current-attempt upload names. Support trusted runner OIDC reusable-workflow claims when GHES lacks native job identity, requiring explicit permission opt-in and usable server claims. Preserve PR-aware default refs for inlined dispatches and verify canonical artifact producers and consumers. Update compiler-threat and safe-output contract specifications; hosted rerun and GHES runtime validation remains outstanding.

Preserve expression-valued step `continue-on-error` options while inspecting reusable-workflow artifacts, including external detector installation and custom jobs. Unbound artifact channels remain optional only when `continue-on-error` is literally `true`.
