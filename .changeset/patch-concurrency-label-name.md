---
"gh-aw": patch
---

Include `github.event.label.name` in the concurrency group for label-triggered workflows (label trigger shorthand and label_command). This prevents cross-label cancellation when multiple labels are added to the same PR or issue simultaneously.
