---
"gh-aw": patch
---

Fixed incremental patch generation for `push_to_pull_request_branch` by falling back to an existing `origin/<branch>` tracking ref when `git fetch` fails, and added integration coverage for the fallback path.
