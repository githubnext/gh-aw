---
"gh-aw": minor
---

Add support for discussion and discussion_comment events in command trigger

The command trigger now recognizes GitHub Discussions events, allowing agentic workflows to respond to `/mention` commands in discussions just like they do for issues and pull requests. This includes support for both `discussion` (when a discussion is created or edited) and `discussion_comment` (when a comment on a discussion is created or edited) events.
