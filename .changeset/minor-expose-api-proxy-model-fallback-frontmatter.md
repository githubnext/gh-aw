---
"gh-aw": minor
---

Expose `engine.firewall.apiProxy.modelFallback` in the compiler frontmatter so BYOK Azure OpenAI users can disable the middle-power fallback strategy that rewrites deployment names and causes HTTP 404 `DeploymentNotFound` errors.

Example usage:

```yaml
engine:
  id: copilot
  firewall:
    apiProxy:
      modelFallback:
        enabled: false
```
