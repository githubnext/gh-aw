---
"gh-aw": patch
---

Ensure `clean_git_credentials.sh` recursively discovers every `.git/config` under the workspace and `/tmp/`, deduplicates the list, and reuses a helper when cleaning each file.
