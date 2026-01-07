---
"gh-aw": patch
---

Add domain blocklist support via `--block-domains` flag.

This change adds support for specifying blocked domains in workflow frontmatter and passes the `--block-domains` flag to Copilot/Claude/Codex engines during compilation. Includes parser updates, unit and integration tests, and documentation updates.
