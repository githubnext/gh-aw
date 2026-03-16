---
"gh-aw": patch
---

Add support for the `label_command` trigger so workflows can run when a configured label is added to an issue, pull request, or discussion. The activation job now removes the triggering label at startup and exposes `needs.activation.outputs.label_command` for downstream use.
