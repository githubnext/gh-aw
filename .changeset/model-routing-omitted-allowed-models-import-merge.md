---
"gh-aw": patch
---

Treat an omitted `engine.model-routing.allowed-models` as "any model" when merging routing settings from imported workflows. If the main workflow or any import omits the list, the router no longer gets a shortlist narrowed to the other workflows' entries; when every workflow lists models, the lists are still combined.
