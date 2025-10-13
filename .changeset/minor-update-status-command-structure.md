---
"gh-aw": minor
---

Update status command JSON output structure

The status command with --json flag now:
- Replaces `agent` field with `engine_id` for clarity
- Removes `frontmatter` and `prompt` fields
- Adds `on` field from workflow frontmatter to show trigger configuration
