---
"gh-aw": minor
---

Changed the workflow frontmatter field `engine` to require an object instead of a string.

## Codemod

If you have workflows using the old string format for the `engine` field:

```yaml
engine: copilot
```

Update them to use the new object format:

```yaml
engine:
  id: copilot
```

This change applies to all workflows using the `engine` field in their frontmatter.
