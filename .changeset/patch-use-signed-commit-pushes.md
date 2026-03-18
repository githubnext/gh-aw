---
"gh-aw": patch
---

Replace direct `git push` with GraphQL commit replay so commits pushed by `push_to_pull_request_branch` and `create_pull_request` are GitHub-signed, with fallback to `git push` when GraphQL commit creation is unavailable.
