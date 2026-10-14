---
"gh-aw": patch
---

Require `checkout.multi: true` to enable runtime repository lists and their restrictions. Single-repository expressions no longer require a multi-repo allowlist or path prefix; both checkout jobs reject values that do not identify exactly one repository.
