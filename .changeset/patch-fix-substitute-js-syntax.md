---
"gh-aw": patch
---

Fix JavaScript export/invocation bug in placeholder substitution.

Updated the JS substitution helper to export a named async function and
adjusted the generated call site in the compiler to invoke that function.
Recompiled workflows and verified generated scripts pass Node.js syntax
validation.

