---
"gh-aw": patch
---

Add lazy mention resolution with collaborator filtering, assignee support, and a 50-mention limit.

This change introduces a dedicated `resolve_mentions` module that lazily
resolves @-mentions, caches recent collaborators for optimistic resolution,
filters out bots, and adds assignees to known aliases. It also updates
workflows to include author/assignee mentions where appropriate.

