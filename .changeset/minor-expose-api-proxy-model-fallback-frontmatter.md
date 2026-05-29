---
"gh-aw": minor
---

Expose `sandbox.agent.model-fallback` in the compiler frontmatter so BYOK Azure OpenAI users can disable the middle-power fallback behavior that rewrites deployment names and causes HTTP 404 `DeploymentNotFound` errors.

Example usage:

```yaml
sandbox:
  agent:
    model-fallback: false
```
