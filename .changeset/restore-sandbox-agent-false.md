---
"gh-aw": major
---

Restore `sandbox.agent: false` as a supported non-strict mode and require `features.dangerously-disable-sandbox-agent: true` to enable it.

The existing `features.dangerously-disable-sandbox-agent` flag now accepts the required boolean value:

```yaml
features:
  dangerously-disable-sandbox-agent: true
sandbox:
  agent: false
strict: false
```
