---
"gh-aw": patch
---

Upgrade google/jsonschema-go to v0.4.0 with critical bug fixes and new features

This upgrade brings several improvements:
- PropertyOrder feature for deterministic property ordering
- Fixed nullable types for slices and pointers 
- Full Draft-07 support
- JSON marshal consistency fixes

Updated tests to handle v0.4.0's new `Types []string` field for nullable types (nullable slices now use `["null", "array"]` instead of a single `"array"` type string).
