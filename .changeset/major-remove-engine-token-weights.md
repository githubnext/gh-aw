---
"gh-aw": major
---

Removed the deprecated `engine.token-weights` frontmatter field from the schema, compiler, and generated docs.

**⚠️ Breaking Change**: Workflows that still set `engine.token-weights` now fail validation.

**Migration guide:**
- Remove the `engine.token-weights` block from workflow frontmatter
- Use the built-in model cost defaults for run analysis
