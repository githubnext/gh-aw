---
"gh-aw": patch
---

Ensure the substitution step always publishes known `needs.*` expression mappings and automatically rewrites `needs.activation.outputs.{text,title,body}` to `steps.sanitized.outputs.*`, so prompts can keep referencing sanitized activation outputs even when markdown changes during runtime import.
