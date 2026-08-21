---
"gh-aw": patch
---

Reject placeholder secret values such as `null`, `undefined`, and whitespace-only strings during activation secret validation, and log configured secret lengths without exposing secret values.
