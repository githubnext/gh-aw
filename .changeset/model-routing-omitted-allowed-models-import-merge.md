---
"gh-aw": patch
---

Treat an omitted `engine.model-routing.allowed-models` as "any model" when merging routing settings from imported workflows. Among workflows that enable routing for the same engine, if any omits `allowed-models`, the router has no shortlist; otherwise, their lists are combined.
