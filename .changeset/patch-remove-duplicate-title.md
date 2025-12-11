---
"gh-aw": patch
---

Add a JavaScript helper that removes duplicate titles from safe output descriptions and register it in the bundler.

The helper `removeDuplicateTitleFromDescription` is used by create/update scripts for issues, discussions, and pull requests to avoid repeating the title in the description body.
