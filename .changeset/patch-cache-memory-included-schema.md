---
"gh-aw": patch
---

Add cache-memory support to included workflow schema

Update shared workflow frontmatter schema to include cache-memory property definition, matching the main workflow schema. This enables shared workflow files to use cache-memory configuration with boolean, null, and object types including key, docker-image, and retention-days options.
