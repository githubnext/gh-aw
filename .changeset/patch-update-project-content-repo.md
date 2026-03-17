---
"gh-aw": patch
---

Added `target_repo` field to `update_project` safe output for cross-repository content resolution. Organization-level projects can now update fields for items from any configured repository by specifying `target_repo: "owner/repo"` in agent output. Configure `allowed-repos` in frontmatter to control which repositories are permitted.
