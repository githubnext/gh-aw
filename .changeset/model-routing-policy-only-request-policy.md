---
"gh-aw": patch
---

Model routing: `models.allowed` and `models.blocked` are now the only request policy for routed workflows, so built-in agents and sub-agents may use any policy-allowed model. `engine.model-routing.allowed-models` is optional and only limits the router's candidates for the main agent; when omitted, Claude and Codex are limited to their model family, and compilation fails if the policy leaves no model in it. AWF versions before v0.28.33 still require it.
