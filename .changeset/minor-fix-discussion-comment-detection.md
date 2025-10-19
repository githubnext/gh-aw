---
"gh-aw": minor
---

Fix add_comment to auto-detect discussion context and use GraphQL API

**Breaking Change**: Removed the `discussion: true` configuration option. The add_comment safe-output now automatically detects discussion contexts from event types (`discussion` or `discussion_comment`) and uses the appropriate API without requiring explicit configuration.
