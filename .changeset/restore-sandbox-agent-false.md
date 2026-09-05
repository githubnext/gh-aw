---
"gh-aw": major
---

Restore `sandbox.agent: false` as a supported non-strict mode and require `features.dangerously-disable-sandbox: true` to enable it.

**Breaking change:** Replace the previous `features.dangerously-disable-sandbox-agent` justification with the new boolean feature flag:

```yaml
features:
  dangerously-disable-sandbox: true
sandbox:
  agent: false
strict: false
```
