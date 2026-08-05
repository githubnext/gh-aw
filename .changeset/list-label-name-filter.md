---
"gh-aw": patch
---

Add an optional `nameFilter` to the `list_label` mcp-scripts pagination wrapper (and a matching `--name-filter` flag on the `github-labels-query` skill script) so agents that only need a few labels no longer pay for the full repository label set.
