---
"gh-aw": patch
---

Fixed the `copilot` engine to support automatic model selection on plans limited to it (e.g. Copilot Free). Previously, every workflow path injected an explicit `COPILOT_MODEL`, causing the Copilot gateway to reject requests with HTTP 400 for plans that only allow automatic model selection.

Set `model: auto` (or `model: none`) in your workflow frontmatter to skip model pinning and let the Copilot CLI choose the model automatically:

```yaml
engine:
  id: copilot
  model: auto
```

Both `auto` and `none` are pass-through sentinels: they suppress `COPILOT_MODEL` injection so the Copilot CLI performs its own automatic routing, which is the only supported mode on Copilot Free accounts.
