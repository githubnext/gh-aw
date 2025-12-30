---
"gh-aw": patch
---

Track unresolved temporary IDs in safe outputs and perform synthetic
updates once those IDs are resolved. This ensures outputs (issues,
discussions, comments) created with unresolved temporary IDs are
updated to contain final values after resolution.

This is an internal fix to the safe output processing logic and does
not introduce any breaking API changes.

