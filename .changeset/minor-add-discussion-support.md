---
"gh-aw": minor
---

Add discussion support to add_reaction_and_edit_comment.cjs

The workflow script now supports GitHub Discussions events (`discussion` and `discussion_comment`), enabling agentic workflows to add reactions and comments to discussions. This extends the existing functionality that previously only supported issues and pull requests. The implementation uses GraphQL API for all discussion operations and includes comprehensive test coverage.
